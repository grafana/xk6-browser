package chromium

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/grafana/xk6-browser/api"
	"github.com/grafana/xk6-browser/common"
	"github.com/grafana/xk6-browser/k6ext"
	"github.com/grafana/xk6-browser/log"
	"github.com/grafana/xk6-browser/storage"

	k6common "go.k6.io/k6/js/common"
	k6modules "go.k6.io/k6/js/modules"
	k6lib "go.k6.io/k6/lib"

	"github.com/dop251/goja"
)

// Ensure BrowserType implements the api.BrowserType interface.
var _ api.BrowserType = &BrowserType{}

// BrowserType provides methods to launch a Chrome browser instance or connect to an existing one.
// It's the entry point for interacting with the browser.
type BrowserType struct {
	// FIXME: This is only exported because testBrowser needs it. Contexts
	// shouldn't be stored on structs if we can avoid it.
	Ctx       context.Context
	vu        k6modules.VU
	hooks     *common.Hooks
	k6Metrics *k6ext.CustomMetrics
	execPath  string       // path to the Chromium executable
	storage   *storage.Dir // stores temporary data for the extension and user
}

// NewBrowserType registers our custom k6 metrics, creates method mappings on
// the goja runtime, and returns a new Chrome browser type.
func NewBrowserType(vu k6modules.VU) api.BrowserType {
	var (
		rt    = vu.Runtime()
		hooks = common.NewHooks()
	)

	// NOTE: vu.InitEnv() *must* be called from the script init scope,
	// otherwise it will return nil.
	k6m := k6ext.RegisterCustomMetrics(vu.InitEnv().Registry)
	b := BrowserType{
		vu:        vu,
		hooks:     hooks,
		k6Metrics: k6m,
		storage:   &storage.Dir{},
	}
	rt.SetFieldNameMapper(common.NewFieldNameMapper())

	return &b
}

// Connect attaches k6 browser to an existing browser instance.
func (b *BrowserType) Connect(opts goja.Value) {
	rt := b.vu.Runtime()
	k6common.Throw(rt, errors.New("BrowserType.connect() has not been implemented yet"))
}

// ExecutablePath returns the path where the extension expects to find the browser executable.
func (b *BrowserType) ExecutablePath() (execPath string) {
	if b.execPath != "" {
		return b.execPath
	}
	defer func() {
		b.execPath = execPath
	}()

	for _, path := range [...]string{
		// Unix-like
		"headless_shell",
		"headless-shell",
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
		"google-chrome-beta",
		"google-chrome-unstable",
		"/usr/bin/google-chrome",

		// Windows
		"chrome",
		"chrome.exe", // in case PATHEXT is misconfigured
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		filepath.Join(os.Getenv("USERPROFILE"), `AppData\Local\Google\Chrome\Application\chrome.exe`),

		// Mac (from https://commondatastorage.googleapis.com/chromium-browser-snapshots/index.html?prefix=Mac/857950/)
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	} {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}

	return ""
}

func (b *BrowserType) initContext() context.Context {
	ctx := k6ext.WithVU(b.vu.Context(), b.vu)
	ctx = k6ext.WithCustomMetrics(ctx, b.k6Metrics)
	ctx = common.WithHooks(ctx, b.hooks)
	return ctx
}

// Launch allocates a new Chrome browser process and returns a new api.Browser value,
// which can be used for controlling the Chrome browser.
func (b *BrowserType) Launch(opts goja.Value) api.Browser {
	ctx := b.initContext()
	launchOpts := common.NewLaunchOptions()
	if err := launchOpts.Parse(ctx, opts); err != nil {
		k6ext.Panic(ctx, "parsing launch options: %w", err)
	}
	ctx = common.WithLaunchOptions(ctx, launchOpts)

	bp, err := b.launch(ctx, launchOpts)
	if err != nil {
		err = &k6ext.UserFriendlyError{
			Err:     err,
			Timeout: launchOpts.Timeout,
		}
		k6ext.Panic(ctx, "%w", err)
	}

	return bp
}

func (b *BrowserType) launch(ctx context.Context, opts *common.LaunchOptions) (*common.Browser, error) {
	envs := make([]string, 0, len(opts.Env))
	for k, v := range opts.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	logger, err := makeLogger(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("setting up logger: %w", err)
	}

	var (
		flags   = prepareFlags(opts, &(b.vu.State()).Options)
		dataDir = b.storage
	)
	if err := dataDir.Make("", flags["user-data-dir"]); err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	flags["user-data-dir"] = dataDir.Dir

	go func(c context.Context) {
		defer func() {
			if err := dataDir.Cleanup(); err != nil {
				logger.Errorf("BrowserType:Launch", "cleaning up the user data directory: %v", err)
			}
		}()
		// There's a small chance that this might be called
		// if the context is closed by the k6 runtime. To
		// guarantee the cleanup we would need to orchestrate
		// it correctly which https://github.com/grafana/k6/issues/2432
		// will enable once it's complete.
		<-c.Done()
	}(ctx)

	browserProc, err := b.allocate(ctx, opts, flags, envs, dataDir, logger)
	if browserProc == nil {
		return nil, fmt.Errorf("launching browser: %w", err)
	}

	browserProc.AttachLogger(logger)

	// If this context is cancelled we'll initiate an extension wide
	// cancellation and shutdown.
	browserCtx, browserCtxCancel := context.WithCancel(ctx)
	// attach the browser process ID to the context
	// so that we can kill it afterward if it lingers
	// see: k6ext.Panic function.
	browserCtx = k6ext.WithProcessID(browserCtx, browserProc.Pid())
	b.Ctx = browserCtx
	browser, err := common.NewBrowser(browserCtx, browserCtxCancel,
		browserProc, opts, logger)
	if err != nil {
		return nil, fmt.Errorf("launching browser: %w", err)
	}

	return browser, nil
}

// LaunchPersistentContext launches the browser with persistent storage.
func (b *BrowserType) LaunchPersistentContext(userDataDir string, opts goja.Value) api.Browser {
	rt := b.vu.Runtime()
	k6common.Throw(rt, errors.New("BrowserType.LaunchPersistentContext(userDataDir, opts) has not been implemented yet"))
	return nil
}

// Name returns the name of this browser type.
func (b *BrowserType) Name() string {
	return "chromium"
}

// allocate starts a new Chromium browser process and returns it.
func (b *BrowserType) allocate(
	ctx context.Context, opts *common.LaunchOptions,
	flags map[string]interface{}, env []string, dataDir *storage.Dir,
	logger *log.Logger,
) (_ *common.BrowserProcess, rerr error) {
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer func() {
		if rerr != nil {
			cancel()
		}
	}()

	args, err := parseArgs(flags)
	if err != nil {
		return nil, err
	}

	path := opts.ExecutablePath
	if path == "" {
		path = b.ExecutablePath()
	}

	cmd, stdout, err := execute(ctx, path, args, env, dataDir, logger)
	if err != nil {
		return nil, err
	}

	wsURL, err := parseDevToolsURL(ctx, stdout)
	if err != nil {
		return nil, fmt.Errorf("getting DevTools URL: %w", err)
	}

	return common.NewBrowserProcess(ctx, cancel, cmd.Process, wsURL, dataDir), nil
}

// parseArgs parses command-line arguments and returns them.
func parseArgs(flags map[string]interface{}) ([]string, error) {
	// Build command line args list
	var args []string
	for name, value := range flags {
		switch value := value.(type) {
		case string:
			args = append(args, fmt.Sprintf("--%s=%s", name, value))
		case bool:
			if value {
				args = append(args, fmt.Sprintf("--%s", name))
			}
		default:
			return nil, fmt.Errorf(`invalid browser command line flag: "%s=%v"`, name, value)
		}
	}
	if _, ok := flags["no-sandbox"]; !ok && os.Getuid() == 0 {
		// Running as root, for example in a Linux container. Chromium
		// needs --no-sandbox when running as root, so make that the
		// default, unless the user set "no-sandbox": false.
		args = append(args, "--no-sandbox")
	}
	if _, ok := flags["remote-debugging-port"]; !ok {
		args = append(args, "--remote-debugging-port=0")
	}

	// Force the first page to be blank, instead of the welcome page;
	// --no-first-run doesn't enforce that.
	// args = append(args, "about:blank")
	// args = append(args, "--no-startup-window")
	return args, nil
}

func prepareFlags(lopts *common.LaunchOptions, k6opts *k6lib.Options) map[string]interface{} {
	// After Puppeteer's and Playwright's default behavior.
	f := map[string]interface{}{
		"disable-background-networking":                      true,
		"enable-features":                                    "NetworkService,NetworkServiceInProcess",
		"disable-background-timer-throttling":                true,
		"disable-backgrounding-occluded-windows":             true,
		"disable-breakpad":                                   true,
		"disable-client-side-phishing-detection":             true,
		"disable-component-extensions-with-background-pages": true,
		"disable-default-apps":                               true,
		"disable-dev-shm-usage":                              true,
		"disable-extensions":                                 true,
		//nolint:lll
		"disable-features":                 "ImprovedCookieControls,LazyFrameLoading,GlobalMediaControls,DestroyProfileOnBrowserClose,MediaRouter,AcceptCHFrame",
		"disable-hang-monitor":             true,
		"disable-ipc-flooding-protection":  true,
		"disable-popup-blocking":           true,
		"disable-prompt-on-repost":         true,
		"disable-renderer-backgrounding":   true,
		"disable-sync":                     true,
		"force-color-profile":              "srgb",
		"metrics-recording-only":           true,
		"no-first-run":                     true,
		"safebrowsing-disable-auto-update": true,
		"enable-automation":                true,
		"password-store":                   "basic",
		"use-mock-keychain":                true,
		"no-service-autorun":               true,

		"no-startup-window":           true,
		"no-default-browser-check":    true,
		"no-sandbox":                  true,
		"headless":                    lopts.Headless,
		"auto-open-devtools-for-tabs": lopts.Devtools,
		"window-size":                 fmt.Sprintf("%d,%d", 800, 600),
	}
	if runtime.GOOS == "darwin" {
		f["enable-use-zoom-for-dsf"] = false
	}
	if lopts.Headless {
		f["hide-scrollbars"] = true
		f["mute-audio"] = true
		f["blink-settings"] = "primaryHoverType=2,availableHoverTypes=2,primaryPointerType=4,availablePointerTypes=4"
	}

	setFlagsFromArgs(f, lopts.Args)
	setFlagsFromK6Options(f, k6opts)

	return f
}

// setFlagsFromArgs fills flags by parsing the args slice.
// This is used for passing the "arg=value" arguments along with other launch options
// when launching a new Chrome browser.
func setFlagsFromArgs(flags map[string]interface{}, args []string) {
	var argname, argval string
	for _, arg := range args {
		pair := strings.SplitN(arg, "=", 2)
		argname, argval = strings.TrimSpace(pair[0]), ""
		if len(pair) > 1 {
			argval = common.TrimQuotes(strings.TrimSpace(pair[1]))
		}
		flags[argname] = argval
	}
}

// setFlagsFromK6Options adds additional data to flags considering the k6 options.
// Such as: "host-resolver-rules" for blocking requests.
func setFlagsFromK6Options(flags map[string]interface{}, k6opts *k6lib.Options) {
	if k6opts == nil {
		return
	}

	hostResolver := []string{}
	if currHostResolver, ok := flags["host-resolver-rules"]; ok {
		hostResolver = append(hostResolver, fmt.Sprintf("%s", currHostResolver))
	}

	for k, v := range k6opts.Hosts {
		hostResolver = append(hostResolver, fmt.Sprintf("MAP %s %s", k, v))
	}

	if len(hostResolver) > 0 {
		sort.Strings(hostResolver)
		flags["host-resolver-rules"] = strings.Join(hostResolver, ",")
	}
}

func execute(
	ctx context.Context, path string, args, env []string, dataDir *storage.Dir, logger *log.Logger,
) (*exec.Cmd, io.Reader, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	killAfterParent(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("%w", err)
	}
	cmd.Stderr = cmd.Stdout

	// Set up environment variable for process
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	// We must start the cmd before calling cmd.Wait, as otherwise the two
	// can run into a data race.
	err = cmd.Start()
	if os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("file does not exist: %s", path)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("%w", err)
	}
	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("%w", ctx.Err())
	}
	go func() {
		// TODO: How to handle these errors?
		defer func() {
			if err := dataDir.Cleanup(); err != nil {
				logger.Errorf("BrowserType:Close", "cleaning up the user data directory: %v", err)
			}
		}()

		if err := cmd.Wait(); err != nil {
			logErr := logger.Errorf
			if s := err.Error(); strings.Contains(s, "signal: killed") || strings.Contains(s, "exit status 1") {
				// The browser process is killed when the context is cancelled
				// after a k6 iteration ends, so silence the log message until
				// we can stop it gracefully. See #https://github.com/grafana/xk6-browser/issues/423
				logErr = logger.Debugf
			}
			logErr(
				"browser", "process with PID %d unexpectedly ended: %v",
				cmd.Process.Pid, err,
			)
		}
	}()

	return cmd, stdout, nil
}

// parseDevToolsURL grabs the websocket address from chrome's output and returns it.
func parseDevToolsURL(ctx context.Context, rc io.Reader) (wsURL string, _ error) {
	type result struct {
		devToolsURL string
		err         error
	}
	c := make(chan result, 1)
	go func() {
		const prefix = "DevTools listening on "

		scanner := bufio.NewScanner(rc)
		for scanner.Scan() {
			if s := scanner.Text(); strings.HasPrefix(s, prefix) {
				c <- result{
					strings.TrimPrefix(strings.TrimSpace(s), prefix),
					nil,
				}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			c <- result{"", err}
		}
	}()
	select {
	case r := <-c:
		return r.devToolsURL, r.err
	case <-ctx.Done():
		return "", fmt.Errorf("%w", ctx.Err())
	}
}

// makeLogger makes and returns an extension wide logger.
func makeLogger(ctx context.Context, launchOpts *common.LaunchOptions) (*log.Logger, error) {
	var (
		k6Logger            = k6ext.GetVU(ctx).State().Logger
		reCategoryFilter, _ = regexp.Compile(launchOpts.LogCategoryFilter)
		logger              = log.New(k6Logger, launchOpts.Debug, reCategoryFilter)
	)

	// set the log level from the launch options (usually from a script's options).
	if launchOpts.Debug {
		_ = logger.SetLevel("debug")
	}
	if el, ok := os.LookupEnv("XK6_BROWSER_LOG"); ok {
		if logger.SetLevel(el) != nil {
			return nil, fmt.Errorf(
				"invalid log level %q, should be one of: panic, fatal, error, warn, warning, info, debug, trace",
				el,
			)
		}
	}
	if _, ok := os.LookupEnv("XK6_BROWSER_CALLER"); ok {
		logger.ReportCaller()
	}

	return logger, nil
}
