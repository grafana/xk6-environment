# xk6-environment

This work is done as quick PoC for Hackathon project, Aug 2023 ([`#hackathon-2023-08-k6-ephemeral`](https://raintank-corp.slack.com/archives/C05K5HF0YCF)).

This is an adaptation of CLI tool [k6-environment](https://github.com/grafana/k6-environment) for xk6 extension, following [this proposal](https://github.com/grafana/k6-environment/blob/main/extension-proposal.js). For main info see README of k6-environment.

## How to

To build:
```
xk6 build --with xk6-environment=.
```

To run:
```
./k6 run sample.js
```

where sample.js contains the following:
```js
import environment from 'k6/x/environment';

const TestWithEnvironment = new environment.New({
    // Location of the test with environment
    source: "examples/testapi-k6/",
    includeGrafana: true, // is ignored for now

    criteria: {
        // run until the test is finished successfully
        test: "finished",
    },

    timeout: "24h", // is ignored for now
})

export function setup() {
    // to ensure execution happens only once, we run creation of environment in setup
    TestWithEnvironment.create();
}

export default function () {
    TestWithEnvironment.runTest();
}

export function teardown() {
    TestWithEnvironment.delete();
}
```

## Changes from k6-environment

This repo is a copy-paste of k6-environment, with the following changes:
- `register.go` instead of `main.go`
- changes to `pkg/environment`: see comments there

Otherwise, xk6-environment and k6-environment are identical. Which means we can have both CLI and xk6 extension with the same approach :tada: