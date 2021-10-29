import { check, group } from "k6";
import Ajv from 'https://jslib.k6.io/ajv/6.12.5/index.js';

export class FunkBrokenChainException extends Error {
  constructor(message) {
    super(message);
    this.brokenChain = true;
    this.name = this.constructor.name;
    if (typeof Error.captureStackTrace === 'function') {
      Error.captureStackTrace(this, this.constructor)
    } else {
      this.stack = new Error(message).stack
    }
  }
}

class Funk {
  constructor() {
    this.leftHandValue = null;     // resp.status
    this.leftHandValueName = null; // "my status"
    this.rightHandValue = null;    // 200
    this.chainBroken = false;
    this.printedBrokenChainWarning = false; // print only one warning.
    this.ajv = new Ajv();
  };

  as(name) {
    this.leftHandValueName = name;
    return this
  }

  _brokenChainCheck() {
    if (this.chainBroken) {
      if (!this.printedBrokenChainWarning) {
        console.warn("This check has been aborted because the previous check in the chain has failed");
        this.printedBrokenChainWarning = true;
      }
      return true
    }
    return false;
  }

  _recordCheck(checkName, isSuccessful, value) {
    if (value !== undefined) {
      check(null, {
        [checkName]: isSuccessful
      }, {
        value: value
      });
    }
    else {
      check(null, {
        [checkName]: isSuccessful
      });
    }
  }

  _breakTheChain() {
    this.chainBroken = true;
    throw new FunkBrokenChainException("Chain broke, skipping this check");
  }

  _leftHandValueIsHttpResponse(calee) {
    // TODO: I don't know how to check that this.leftHandValue is of type HttpResponse 
    if (this.leftHandValue && this.leftHandValue.hasOwnProperty('request')) {
      return true;
    }
    else {
      console.error(`The object passed to expect/and for ${calee} isn't a k6 HttpResponse. Aborting the check.`);
      this._breakTheChain();
      this.printedBrokenChainWarning = true;
      return false
    }
  }

  toMatchAPISchema(schema) {
    let validate = this.ajv.compile(schema);
    let data = this.leftHandValue;

    let is_valid = validate(data, schema);
    let name = this.leftHandValueName || '';

    check(is_valid, {
      [`${name} schema validation`]: (is_valid) => is_valid
    });

    // optional. It records specific error messages as checks.
    if (!is_valid) {
      // console.error(JSON.stringify(validate.errors));

      validate.errors.forEach(error => {
        check(is_valid, {
          [`${name} ${error.dataPath} ${error.message}`]: (is_valid) => is_valid
        });
      });
      this._breakTheChain();
    }

    return this;
  }

  toHaveValidJson() {
    if (this._brokenChainCheck()) return this;
    if (!this._leftHandValueIsHttpResponse("toHaveValidJson")) return this;

    let resp = this.leftHandValue;

    let checkIsSuccessful = true;
    try {
      resp.json();
    }
    catch (e) {
      checkIsSuccessful = false;
    }

    let checkName = `${resp.request.url} has valid json response`
    this._recordCheck(checkName, checkIsSuccessful);
    if (!checkIsSuccessful) this._breakTheChain();

    return this
  }

  toEqual(rhv) {
    if (this._brokenChainCheck()) return this;
    this.rightHandValue = rhv;

    let checkName = `${this.leftHandValue} is ${this.rightHandValue}`;

    let checkIsSuccessful = this.leftHandValue === this.rightHandValue;

    if (this.leftHandValueName) {
      checkName = `${this.leftHandValueName} is ${this.leftHandValue}.`;

      if (!checkIsSuccessful) {
        checkName += ` Expected '${this.rightHandValue}'`;
      }
    }

    this._recordCheck(checkName, checkIsSuccessful, this.rightHandValue);

    if (!checkIsSuccessful) this._breakTheChain();

    return this;
  }

  toBeGreaterThan(rhv) {
    if (this._brokenChainCheck()) return this;

    this.rightHandValue = rhv;

    let checkName = `${this.leftHandValueName || this.leftHandValue} is greater than ${this.rightHandValue}`;

    let checkIsSuccessful = this.leftHandValue > this.rightHandValue;

    this._recordCheck(checkName, checkIsSuccessful, this.leftHandValue);

    if (!checkIsSuccessful) this._breakTheChain();

    return this;
  }

  toBeGreaterThanOrEqual(rhv) {
    if (this._brokenChainCheck()) return this;

    this.rightHandValue = rhv;

    let checkName = `${this.leftHandValueName || this.leftHandValue} is greater or equal to ${this.rightHandValue}`;

    let checkIsSuccessful = this.leftHandValue >= this.rightHandValue;

    this._recordCheck(checkName, checkIsSuccessful, this.leftHandValue);

    if (!checkIsSuccessful) this._breakTheChain();

    return this;
  }
  toBeLessThan(rhv) {
    if (this._brokenChainCheck()) return this;

    this.rightHandValue = rhv;

    let checkName = `${this.leftHandValueName || this.leftHandValue} is less than ${this.rightHandValue}`;

    let checkIsSuccessful = this.leftHandValue < this.rightHandValue;

    this._recordCheck(checkName, checkIsSuccessful, this.leftHandValue);

    if (!checkIsSuccessful) this._breakTheChain();

    return this;
  }
  toBeLessThanOrEqual(rhv) {
    if (this._brokenChainCheck()) return this;

    this.rightHandValue = rhv;

    let checkName = `${this.leftHandValueName || this.leftHandValue} is less or equal to ${this.rightHandValue}`;

    let checkIsSuccessful = this.leftHandValue <= this.rightHandValue;

    this._recordCheck(checkName, checkIsSuccessful, this.leftHandValue);

    if (!checkIsSuccessful) this._breakTheChain();

    return this;
  }

  toBeTruthy() {
    if (this._brokenChainCheck()) return this;

    let checkName = `${this.leftHandValueName || this.leftHandValue} is truthy.`;

    let checkIsSuccessful = this.leftHandValue ? true : false;

    this._recordCheck(checkName, checkIsSuccessful, this.leftHandValue);

    if (!checkIsSuccessful) this._breakTheChain();

    return this;
  }

  toBeBetween(from, to) {
    if (this._brokenChainCheck()) return this;

    this.rightHandValue = `${from} - ${to}`;

    let checkName = `${this.leftHandValueName || this.leftHandValue} is between ${this.rightHandValue}`;

    let checkIsSuccessful = this.leftHandValue >= from && this.leftHandValue <= to;

    this._recordCheck(checkName, checkIsSuccessful, this.leftHandValue);

    if (!checkIsSuccessful) this._breakTheChain();

    return this;
  }

  and(lhv) { // same as expect() but chained.
    if (this._brokenChainCheck()) return this;
    this.leftHandValue = lhv;
    this.leftHandValueName = null; // clearing the previous .as()
    return this;
  }

}

let expect = function (value1) {
  let state = new Funk();
  state.leftHandValue = value1;
  return state;
};

function handleUnexpectedException(e, testName) {
  console.error(`Exception raised in test "${testName}". Failing the test and continuing. \n${e}`)

  check(null, {
    [`Exception raised "${e}"`]: false
  });
}

let describe = function (testName, callback) {
  let t = {
    expect,
  };

  let success = true;

  group(testName, () => {
    try {
      callback(t);
      success = true;
    }
    catch (e) {
      if (e.brokenChain) {
        success = false;
      }
      else {
        success = false;
        handleUnexpectedException(e, testName)
      }
    }
  });
  return success;
};


export {
  describe,
}