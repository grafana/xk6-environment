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
 *   let instance = new Environment("Wonderful World")
 *   console.log(instance.greeting)
 * }
 * ```
 */
export declare class Environment {
  /**
   * Create a new Environment instance.
   *
   * @param name name of the environment
   * @param type implementation for the environment (only "vcluster" for now)
   * @param initFolder folder containing base manifests to apply on initialization of environment
   */
  constructor(params: object);

  init();
  delete();

  // apply(files: string[]); arrays are not supported by Tygor :sweat_smile:
  apply(file: string);
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
   */
  wait(condition: object, opts?: object);

  /**
   * getN is a substitute for get(), hopefully temporary. See tygor's roadmap.
   * @param type 
   * @param opts 
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
