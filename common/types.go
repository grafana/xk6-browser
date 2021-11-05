/*
 *
 * xk6-browser - a browser automation extension for k6
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/dop251/goja"
	k6common "go.k6.io/k6/js/common"
)

// ColorScheme represents a browser color scheme
type ColorScheme string

// Valid color schemes
const (
	ColorSchemeLight        ColorScheme = "light"
	ColorSchemeDark         ColorScheme = "dark"
	ColorSchemeNoPreference ColorScheme = "no-preference"
)

// Credentials holds HTTP authentication credentials
type Credentials struct {
	Username string
	Password string
}

// DOMElementState represents a DOM element state
type DOMElementState int

// Valid DOM element states
const (
	DOMElementStateAttached DOMElementState = iota
	DOMElementStateDetached
	DOMElementStateVisible
	DOMElementStateHidden
)

func (s DOMElementState) String() string {
	return DOMElementStateToString[s]
}

var DOMElementStateToString = map[DOMElementState]string{
	DOMElementStateAttached: "attached",
	DOMElementStateDetached: "detached",
	DOMElementStateVisible:  "visible",
	DOMElementStateHidden:   "hidden",
}

var DOMElementStateToID = map[string]DOMElementState{
	"attached": DOMElementStateAttached,
	"detached": DOMElementStateDetached,
	"visible":  DOMElementStateVisible,
	"hidden":   DOMElementStateHidden,
}

// MarshalJSON marshals the enum as a quoted JSON string
func (s DOMElementState) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(DOMElementStateToString[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted JSON string to the enum value
func (s *DOMElementState) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value.
	*s = DOMElementStateToID[j]
	return nil
}

type EmulatedSize struct {
	Viewport *Viewport
	Screen   *Screen
}

type Geolocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accurracy float64 `json:"accurracy"`
}

type LifecycleEvent int

const (
	LifecycleEventLoad LifecycleEvent = iota
	LifecycleEventDOMContentLoad
	LifecycleEventNetworkIdle
)

func (l LifecycleEvent) String() string {
	return LifecycleEventToString[l]
}

var LifecycleEventToString = map[LifecycleEvent]string{
	LifecycleEventLoad:           "load",
	LifecycleEventDOMContentLoad: "domcontentloaded",
	LifecycleEventNetworkIdle:    "networkidle",
}

var LifecycleEventToID = map[string]LifecycleEvent{
	"load":             LifecycleEventLoad,
	"domcontentloaded": LifecycleEventDOMContentLoad,
	"networkidle":      LifecycleEventNetworkIdle,
}

// MarshalJSON marshals the enum as a quoted JSON string
func (l LifecycleEvent) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(LifecycleEventToString[l])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmarshals a quoted JSON string to the enum value
func (l *LifecycleEvent) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value.
	*l = LifecycleEventToID[j]
	return nil
}

type MediaType string

const (
	MediaTypeScreen MediaType = "screen"
	MediaTypePrint  MediaType = "print"
)

type PollingType int

const (
	PollingRaf PollingType = iota
	PollingMutation
	PollingInterval
)

func (p PollingType) String() string {
	return PollingTypeToString[p]
}

var PollingTypeToString = map[PollingType]string{
	PollingRaf:      "raf",
	PollingMutation: "mutation",
	PollingInterval: "interval",
}

var PollingTypeToID = map[string]PollingType{
	"raf":      PollingRaf,
	"mutation": PollingMutation,
	"interval": PollingInterval,
}

// MarshalJSON marshals the enum as a quoted JSON string
func (p PollingType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(PollingTypeToString[p])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted JSON string to the enum value
func (p *PollingType) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value.
	*p = PollingTypeToID[j]
	return nil
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ReducedMotion represents a browser reduce-motion setting
type ReducedMotion string

// Valid reduce-motion options
const (
	ReducedMotionReduce       ReducedMotion = "reduce"
	ReducedMotionNoPreference ReducedMotion = "no-preference"
)

type ResourceTiming struct {
	StartTime             float64 `json:"startTime"`
	DomainLookupStart     float64 `json:"domainLookupStart"`
	DomainLookupEnd       float64 `json:"domainLookupEnd"`
	ConnectStart          float64 `json:"connectStart"`
	SecureConnectionStart float64 `json:"secureConnectionStart"`
	ConnectEnd            float64 `json:"connectEnd"`
	RequestStart          float64 `json:"requestStart"`
	ResponseStart         float64 `json:"responseStart"`
	ResponseEnd           float64 `json:"responseEnd"`
}

// Viewport represents a device screen
type Screen struct {
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

type SelectOption struct {
	Value *string `json:"value"`
	Label *string `json:"label"`
	Index *int64  `json:"index"`
}

type Size struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Viewport represents a page viewport
type Viewport struct {
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

func NewCredentials() *Credentials {
	return &Credentials{}
}

func (c *Credentials) Parse(ctx context.Context, credentials goja.Value) error {
	rt := k6common.GetRuntime(ctx)
	if credentials != nil && !goja.IsUndefined(credentials) && !goja.IsNull(credentials) {
		credentials := credentials.ToObject(rt)
		for _, k := range credentials.Keys() {
			switch k {
			case "username":
				c.Username = credentials.Get(k).String()
			case "password":
				c.Password = credentials.Get(k).String()
			}
		}
	}
	return nil
}

func NewEmulatedSize(viewport *Viewport, screen *Screen) *EmulatedSize {
	return &EmulatedSize{
		Viewport: viewport,
		Screen:   screen,
	}
}

func NewGeolocation() *Geolocation {
	return &Geolocation{}
}

func (g *Geolocation) Parse(ctx context.Context, opts goja.Value) error {
	rt := k6common.GetRuntime(ctx)
	longitude := 0.0
	latitude := 0.0
	accuracy := 0.0

	if opts != nil && !goja.IsUndefined(opts) && !goja.IsNull(opts) {
		opts := opts.ToObject(rt)
		for _, k := range opts.Keys() {
			switch k {
			case "accuracy":
				accuracy = opts.Get(k).ToFloat()
			case "latitude":
				latitude = opts.Get(k).ToFloat()
			case "longitude":
				longitude = opts.Get(k).ToFloat()
			}
		}
	}

	if longitude < -180 || longitude > 180 {
		return fmt.Errorf(`invalid longitude "%.2f": precondition -180 <= LONGITUDE <= 180 failed`, longitude)
	}
	if latitude < -90 || latitude > 90 {
		return fmt.Errorf(`invalid latitude "%.2f": precondition -90 <= LATITUDE <= 90 failed`, latitude)
	}
	if accuracy < 0 {
		return fmt.Errorf(`invalid accuracy "%.2f": precondition 0 <= ACCURACY failed`, accuracy)
	}

	g.Accurracy = accuracy
	g.Latitude = latitude
	g.Longitude = longitude
	return nil
}

func (s *Screen) Parse(ctx context.Context, screen goja.Value) error {
	rt := k6common.GetRuntime(ctx)
	if screen != nil && !goja.IsUndefined(screen) && !goja.IsNull(screen) {
		screen := screen.ToObject(rt)
		for _, k := range screen.Keys() {
			switch k {
			case "width":
				s.Width = screen.Get(k).ToInteger()
			case "height":
				s.Height = screen.Get(k).ToInteger()
			}
		}
	}
	return nil
}

func (s Size) enclosingIntSize() *Size {
	return &Size{
		Width:  math.Floor(s.Width + 1e-3),
		Height: math.Floor(s.Height + 1e-3),
	}
}

func (s *Size) Parse(ctx context.Context, viewport goja.Value) error {
	rt := k6common.GetRuntime(ctx)
	if viewport != nil && !goja.IsUndefined(viewport) && !goja.IsNull(viewport) {
		viewport := viewport.ToObject(rt)
		for _, k := range viewport.Keys() {
			switch k {
			case "width":
				s.Width = viewport.Get(k).ToFloat()
			case "height":
				s.Height = viewport.Get(k).ToFloat()
			}
		}
	}
	return nil
}

func (v *Viewport) Parse(ctx context.Context, viewport goja.Value) error {
	rt := k6common.GetRuntime(ctx)
	if viewport != nil && !goja.IsUndefined(viewport) && !goja.IsNull(viewport) {
		viewport := viewport.ToObject(rt)
		for _, k := range viewport.Keys() {
			switch k {
			case "width":
				v.Width = viewport.Get(k).ToInteger()
			case "height":
				v.Height = viewport.Get(k).ToInteger()
			}
		}
	}
	return nil
}
