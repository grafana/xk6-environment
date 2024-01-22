import { Environment } from 'k6/x/environment';

const env = new Environment(
    "some-name",
    "vcluster",
    "examples/internal-svc/",
)

export function setup() {
    // to ensure execution happens only once, we run creation of environment in setup
    env.init();
}

export default function () {
    env.apply("pod.yaml");
    env.wait({
        type: "Pod",
        name: "nginx",
        namespace: "default",
        reason: "Started",
    });
}

export function teardown() {
    env.delete();
}