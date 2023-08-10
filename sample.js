import environment from 'k6/x/environment';

const TestWithEnvironment = new environment.New({
    // Location of the test with environment
    source: "examples/testapi-k6",

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