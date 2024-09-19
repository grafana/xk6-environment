/**
 * ˮsummaryˮ
 *
 * @example
 * ```ts
 * import globalEnvironment, { Environment } from "k6/x/environment"
 *
 * export default function () {
 *   console.log(globalEnvironment.greeting)
 *
 *   let instance = new Environment("Wonderful World")
 *   console.log(instance.greeting)
 * }
 * ```
 */
export as namespace environment;

// /** 
//  * The class for basic access to a Kubernetes object.
//  */
// export declare class KubernetesObject {
//   name() string;
//   namespace() string;
// }

/**
 * This is the primary class of the environment extension.
 *
 * @example
 * ```ts
 * import { Environment } from "k6/x/environment"
 *
 * export default function () {
 *   let env = new Environment({
 *     name: "my-env",
 *     implementation: "vcluster",
 *     initFolder: "my-folder-with-manifests/",
 *   })
 * }
 * ```
 */
export declare class Environment {
  /**
   * Defines a new Environment instance.
   *
   * @param name name of the environment
   * @param implementation implementation for the environment (only "vcluster" for now)
   * @param initFolder optional, a folder containing base manifests to apply on initialization of environment
   */
  constructor(params: object);

  /**
   * init creates an Environment as defined in constructor.
   */
  init();
  /**
   * delete removes an existing Environment.
   */
  delete();

  // consider this definition
  // apply(files: string[]); arrays are not supported by Tygor yet

  /**
   * apply reads the contents of the file and applies them to the virtual cluster.
   * @param file is expected to be a readable yaml file (Kubernetes manifest).
   */
  apply(file: string);

  /**
   * applySpec applies the spec to the virtual cluster.
   * @param spec is expected to be a yaml manifest.
   */
  applySpec(spec: string); // we have to use a diff name here: method overload is not supported

  /**
   * `wait` method blocks execution of the test iteration until a certain condition 
   * is reached or until a timeout. There are 3 major types of conditions now:
   * 
   * 1. Wait until a given Kubernetes event.
   * 
   * 2. Wait until a given `.status.conditions[]` reaches a given value.
   * 
   * 3. Wait until a custom field in `.status` reaches a given value.
   * 
   * @param condition describes the wait condition itself. It should have name, namespace, kind fields.
   * It can be configured with fields: 1) "reason" to wait for Kubernetes event, 2) "condition\_type" and "value", 
   * to wait for `.status.conditions[]`, 3) "status\_key" and "status\_value" to wait for custom `.status` value. 
   * @param opts optional configuration of timeout and interval (defaults are 1h and 2s), for how
   * often to perfrom a check of wait condition.
   */
  wait(condition: object, opts?: object);

  /**
   * getN is a substitute for get(), hopefully temporary. See [tygor's](https://github.com/szkiba/tygor) roadmap about support for arrays.
   * @param type is a kind of resource (currently only "pods" are supported).
   * @param opts optional parameteters for the resource, like namespace and labels.
   */
  getN(type: string, opts?: object): number;

  // TODO:
  // list(resource: string, namespace: string);
  // delete();
}

/** Default Environment instance. */
declare const defaultEnvironment: Environment;

/** Default Environment instance. */
export default defaultEnvironment;
