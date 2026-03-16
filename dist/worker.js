var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __name = (target, value) => __defProp(target, "name", { value, configurable: true });
var __esm = (fn, res) => function __init() {
  return fn && (res = (0, fn[__getOwnPropNames(fn)[0]])(fn = 0)), res;
};
var __commonJS = (cb, mod) => function __require() {
  return mod || (0, cb[__getOwnPropNames(cb)[0]])((mod = { exports: {} }).exports, mod), mod.exports;
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
  // If the importer is in node compatibility mode or this is not an ESM
  // file that has been converted to a CommonJS file using a Babel-
  // compatible transform (i.e. "__esModule" has not been set), then set
  // "default" to the CommonJS "module.exports" for node compatibility.
  isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
  mod
));

// node_modules/unenv/dist/runtime/_internal/utils.mjs
// @__NO_SIDE_EFFECTS__
function createNotImplementedError(name) {
  return new Error(`[unenv] ${name} is not implemented yet!`);
}
// @__NO_SIDE_EFFECTS__
function notImplemented(name) {
  const fn = /* @__PURE__ */ __name(() => {
    throw /* @__PURE__ */ createNotImplementedError(name);
  }, "fn");
  return Object.assign(fn, { __unenv__: true });
}
// @__NO_SIDE_EFFECTS__
function notImplementedClass(name) {
  return class {
    __unenv__ = true;
    constructor() {
      throw new Error(`[unenv] ${name} is not implemented yet!`);
    }
  };
}
var init_utils = __esm({
  "node_modules/unenv/dist/runtime/_internal/utils.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    __name(createNotImplementedError, "createNotImplementedError");
    __name(notImplemented, "notImplemented");
    __name(notImplementedClass, "notImplementedClass");
  }
});

// node_modules/unenv/dist/runtime/node/internal/perf_hooks/performance.mjs
var _timeOrigin, _performanceNow, nodeTiming, PerformanceEntry, PerformanceMark, PerformanceMeasure, PerformanceResourceTiming, PerformanceObserverEntryList, Performance, PerformanceObserver, performance;
var init_performance = __esm({
  "node_modules/unenv/dist/runtime/node/internal/perf_hooks/performance.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    init_utils();
    _timeOrigin = globalThis.performance?.timeOrigin ?? Date.now();
    _performanceNow = globalThis.performance?.now ? globalThis.performance.now.bind(globalThis.performance) : () => Date.now() - _timeOrigin;
    nodeTiming = {
      name: "node",
      entryType: "node",
      startTime: 0,
      duration: 0,
      nodeStart: 0,
      v8Start: 0,
      bootstrapComplete: 0,
      environment: 0,
      loopStart: 0,
      loopExit: 0,
      idleTime: 0,
      uvMetricsInfo: {
        loopCount: 0,
        events: 0,
        eventsWaiting: 0
      },
      detail: void 0,
      toJSON() {
        return this;
      }
    };
    PerformanceEntry = class {
      static {
        __name(this, "PerformanceEntry");
      }
      __unenv__ = true;
      detail;
      entryType = "event";
      name;
      startTime;
      constructor(name, options) {
        this.name = name;
        this.startTime = options?.startTime || _performanceNow();
        this.detail = options?.detail;
      }
      get duration() {
        return _performanceNow() - this.startTime;
      }
      toJSON() {
        return {
          name: this.name,
          entryType: this.entryType,
          startTime: this.startTime,
          duration: this.duration,
          detail: this.detail
        };
      }
    };
    PerformanceMark = class PerformanceMark2 extends PerformanceEntry {
      static {
        __name(this, "PerformanceMark");
      }
      entryType = "mark";
      constructor() {
        super(...arguments);
      }
      get duration() {
        return 0;
      }
    };
    PerformanceMeasure = class extends PerformanceEntry {
      static {
        __name(this, "PerformanceMeasure");
      }
      entryType = "measure";
    };
    PerformanceResourceTiming = class extends PerformanceEntry {
      static {
        __name(this, "PerformanceResourceTiming");
      }
      entryType = "resource";
      serverTiming = [];
      connectEnd = 0;
      connectStart = 0;
      decodedBodySize = 0;
      domainLookupEnd = 0;
      domainLookupStart = 0;
      encodedBodySize = 0;
      fetchStart = 0;
      initiatorType = "";
      name = "";
      nextHopProtocol = "";
      redirectEnd = 0;
      redirectStart = 0;
      requestStart = 0;
      responseEnd = 0;
      responseStart = 0;
      secureConnectionStart = 0;
      startTime = 0;
      transferSize = 0;
      workerStart = 0;
      responseStatus = 0;
    };
    PerformanceObserverEntryList = class {
      static {
        __name(this, "PerformanceObserverEntryList");
      }
      __unenv__ = true;
      getEntries() {
        return [];
      }
      getEntriesByName(_name, _type) {
        return [];
      }
      getEntriesByType(type) {
        return [];
      }
    };
    Performance = class {
      static {
        __name(this, "Performance");
      }
      __unenv__ = true;
      timeOrigin = _timeOrigin;
      eventCounts = /* @__PURE__ */ new Map();
      _entries = [];
      _resourceTimingBufferSize = 0;
      navigation = void 0;
      timing = void 0;
      timerify(_fn, _options) {
        throw createNotImplementedError("Performance.timerify");
      }
      get nodeTiming() {
        return nodeTiming;
      }
      eventLoopUtilization() {
        return {};
      }
      markResourceTiming() {
        return new PerformanceResourceTiming("");
      }
      onresourcetimingbufferfull = null;
      now() {
        if (this.timeOrigin === _timeOrigin) {
          return _performanceNow();
        }
        return Date.now() - this.timeOrigin;
      }
      clearMarks(markName) {
        this._entries = markName ? this._entries.filter((e3) => e3.name !== markName) : this._entries.filter((e3) => e3.entryType !== "mark");
      }
      clearMeasures(measureName) {
        this._entries = measureName ? this._entries.filter((e3) => e3.name !== measureName) : this._entries.filter((e3) => e3.entryType !== "measure");
      }
      clearResourceTimings() {
        this._entries = this._entries.filter((e3) => e3.entryType !== "resource" || e3.entryType !== "navigation");
      }
      getEntries() {
        return this._entries;
      }
      getEntriesByName(name, type) {
        return this._entries.filter((e3) => e3.name === name && (!type || e3.entryType === type));
      }
      getEntriesByType(type) {
        return this._entries.filter((e3) => e3.entryType === type);
      }
      mark(name, options) {
        const entry = new PerformanceMark(name, options);
        this._entries.push(entry);
        return entry;
      }
      measure(measureName, startOrMeasureOptions, endMark) {
        let start;
        let end;
        if (typeof startOrMeasureOptions === "string") {
          start = this.getEntriesByName(startOrMeasureOptions, "mark")[0]?.startTime;
          end = this.getEntriesByName(endMark, "mark")[0]?.startTime;
        } else {
          start = Number.parseFloat(startOrMeasureOptions?.start) || this.now();
          end = Number.parseFloat(startOrMeasureOptions?.end) || this.now();
        }
        const entry = new PerformanceMeasure(measureName, {
          startTime: start,
          detail: {
            start,
            end
          }
        });
        this._entries.push(entry);
        return entry;
      }
      setResourceTimingBufferSize(maxSize) {
        this._resourceTimingBufferSize = maxSize;
      }
      addEventListener(type, listener, options) {
        throw createNotImplementedError("Performance.addEventListener");
      }
      removeEventListener(type, listener, options) {
        throw createNotImplementedError("Performance.removeEventListener");
      }
      dispatchEvent(event) {
        throw createNotImplementedError("Performance.dispatchEvent");
      }
      toJSON() {
        return this;
      }
    };
    PerformanceObserver = class {
      static {
        __name(this, "PerformanceObserver");
      }
      __unenv__ = true;
      static supportedEntryTypes = [];
      _callback = null;
      constructor(callback) {
        this._callback = callback;
      }
      takeRecords() {
        return [];
      }
      disconnect() {
        throw createNotImplementedError("PerformanceObserver.disconnect");
      }
      observe(options) {
        throw createNotImplementedError("PerformanceObserver.observe");
      }
      bind(fn) {
        return fn;
      }
      runInAsyncScope(fn, thisArg, ...args) {
        return fn.call(thisArg, ...args);
      }
      asyncId() {
        return 0;
      }
      triggerAsyncId() {
        return 0;
      }
      emitDestroy() {
        return this;
      }
    };
    performance = globalThis.performance && "addEventListener" in globalThis.performance ? globalThis.performance : new Performance();
  }
});

// node_modules/unenv/dist/runtime/node/perf_hooks.mjs
var init_perf_hooks = __esm({
  "node_modules/unenv/dist/runtime/node/perf_hooks.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    init_performance();
  }
});

// node_modules/@cloudflare/unenv-preset/dist/runtime/polyfill/performance.mjs
var init_performance2 = __esm({
  "node_modules/@cloudflare/unenv-preset/dist/runtime/polyfill/performance.mjs"() {
    init_perf_hooks();
    globalThis.performance = performance;
    globalThis.Performance = Performance;
    globalThis.PerformanceEntry = PerformanceEntry;
    globalThis.PerformanceMark = PerformanceMark;
    globalThis.PerformanceMeasure = PerformanceMeasure;
    globalThis.PerformanceObserver = PerformanceObserver;
    globalThis.PerformanceObserverEntryList = PerformanceObserverEntryList;
    globalThis.PerformanceResourceTiming = PerformanceResourceTiming;
  }
});

// node_modules/unenv/dist/runtime/mock/noop.mjs
var noop_default;
var init_noop = __esm({
  "node_modules/unenv/dist/runtime/mock/noop.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    noop_default = Object.assign(() => {
    }, { __unenv__: true });
  }
});

// node_modules/unenv/dist/runtime/node/console.mjs
import { Writable } from "node:stream";
var _console, _ignoreErrors, _stderr, _stdout, log, info, trace, debug, table, error, warn, createTask, clear, count, countReset, dir, dirxml, group, groupEnd, groupCollapsed, profile, profileEnd, time, timeEnd, timeLog, timeStamp, Console, _times, _stdoutErrorHandler, _stderrErrorHandler;
var init_console = __esm({
  "node_modules/unenv/dist/runtime/node/console.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    init_noop();
    init_utils();
    _console = globalThis.console;
    _ignoreErrors = true;
    _stderr = new Writable();
    _stdout = new Writable();
    log = _console?.log ?? noop_default;
    info = _console?.info ?? log;
    trace = _console?.trace ?? info;
    debug = _console?.debug ?? log;
    table = _console?.table ?? log;
    error = _console?.error ?? log;
    warn = _console?.warn ?? error;
    createTask = _console?.createTask ?? /* @__PURE__ */ notImplemented("console.createTask");
    clear = _console?.clear ?? noop_default;
    count = _console?.count ?? noop_default;
    countReset = _console?.countReset ?? noop_default;
    dir = _console?.dir ?? noop_default;
    dirxml = _console?.dirxml ?? noop_default;
    group = _console?.group ?? noop_default;
    groupEnd = _console?.groupEnd ?? noop_default;
    groupCollapsed = _console?.groupCollapsed ?? noop_default;
    profile = _console?.profile ?? noop_default;
    profileEnd = _console?.profileEnd ?? noop_default;
    time = _console?.time ?? noop_default;
    timeEnd = _console?.timeEnd ?? noop_default;
    timeLog = _console?.timeLog ?? noop_default;
    timeStamp = _console?.timeStamp ?? noop_default;
    Console = _console?.Console ?? /* @__PURE__ */ notImplementedClass("console.Console");
    _times = /* @__PURE__ */ new Map();
    _stdoutErrorHandler = noop_default;
    _stderrErrorHandler = noop_default;
  }
});

// node_modules/@cloudflare/unenv-preset/dist/runtime/node/console.mjs
var workerdConsole, assert, clear2, context, count2, countReset2, createTask2, debug2, dir2, dirxml2, error2, group2, groupCollapsed2, groupEnd2, info2, log2, profile2, profileEnd2, table2, time2, timeEnd2, timeLog2, timeStamp2, trace2, warn2, console_default;
var init_console2 = __esm({
  "node_modules/@cloudflare/unenv-preset/dist/runtime/node/console.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    init_console();
    workerdConsole = globalThis["console"];
    ({
      assert,
      clear: clear2,
      context: (
        // @ts-expect-error undocumented public API
        context
      ),
      count: count2,
      countReset: countReset2,
      createTask: (
        // @ts-expect-error undocumented public API
        createTask2
      ),
      debug: debug2,
      dir: dir2,
      dirxml: dirxml2,
      error: error2,
      group: group2,
      groupCollapsed: groupCollapsed2,
      groupEnd: groupEnd2,
      info: info2,
      log: log2,
      profile: profile2,
      profileEnd: profileEnd2,
      table: table2,
      time: time2,
      timeEnd: timeEnd2,
      timeLog: timeLog2,
      timeStamp: timeStamp2,
      trace: trace2,
      warn: warn2
    } = workerdConsole);
    Object.assign(workerdConsole, {
      Console,
      _ignoreErrors,
      _stderr,
      _stderrErrorHandler,
      _stdout,
      _stdoutErrorHandler,
      _times
    });
    console_default = workerdConsole;
  }
});

// node_modules/wrangler/_virtual_unenv_global_polyfill-@cloudflare-unenv-preset-node-console
var init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console = __esm({
  "node_modules/wrangler/_virtual_unenv_global_polyfill-@cloudflare-unenv-preset-node-console"() {
    init_console2();
    globalThis.console = console_default;
  }
});

// node_modules/unenv/dist/runtime/node/internal/process/hrtime.mjs
var hrtime;
var init_hrtime = __esm({
  "node_modules/unenv/dist/runtime/node/internal/process/hrtime.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    hrtime = /* @__PURE__ */ Object.assign(/* @__PURE__ */ __name(function hrtime2(startTime) {
      const now = Date.now();
      const seconds = Math.trunc(now / 1e3);
      const nanos = now % 1e3 * 1e6;
      if (startTime) {
        let diffSeconds = seconds - startTime[0];
        let diffNanos = nanos - startTime[0];
        if (diffNanos < 0) {
          diffSeconds = diffSeconds - 1;
          diffNanos = 1e9 + diffNanos;
        }
        return [diffSeconds, diffNanos];
      }
      return [seconds, nanos];
    }, "hrtime"), { bigint: /* @__PURE__ */ __name(function bigint() {
      return BigInt(Date.now() * 1e6);
    }, "bigint") });
  }
});

// node_modules/unenv/dist/runtime/node/internal/tty/read-stream.mjs
var ReadStream;
var init_read_stream = __esm({
  "node_modules/unenv/dist/runtime/node/internal/tty/read-stream.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    ReadStream = class {
      static {
        __name(this, "ReadStream");
      }
      fd;
      isRaw = false;
      isTTY = false;
      constructor(fd) {
        this.fd = fd;
      }
      setRawMode(mode) {
        this.isRaw = mode;
        return this;
      }
    };
  }
});

// node_modules/unenv/dist/runtime/node/internal/tty/write-stream.mjs
var WriteStream;
var init_write_stream = __esm({
  "node_modules/unenv/dist/runtime/node/internal/tty/write-stream.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    WriteStream = class {
      static {
        __name(this, "WriteStream");
      }
      fd;
      columns = 80;
      rows = 24;
      isTTY = false;
      constructor(fd) {
        this.fd = fd;
      }
      clearLine(dir3, callback) {
        callback && callback();
        return false;
      }
      clearScreenDown(callback) {
        callback && callback();
        return false;
      }
      cursorTo(x, y, callback) {
        callback && typeof callback === "function" && callback();
        return false;
      }
      moveCursor(dx, dy, callback) {
        callback && callback();
        return false;
      }
      getColorDepth(env2) {
        return 1;
      }
      hasColors(count3, env2) {
        return false;
      }
      getWindowSize() {
        return [this.columns, this.rows];
      }
      write(str, encoding, cb) {
        if (str instanceof Uint8Array) {
          str = new TextDecoder().decode(str);
        }
        try {
          console.log(str);
        } catch {
        }
        cb && typeof cb === "function" && cb();
        return false;
      }
    };
  }
});

// node_modules/unenv/dist/runtime/node/tty.mjs
var init_tty = __esm({
  "node_modules/unenv/dist/runtime/node/tty.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    init_read_stream();
    init_write_stream();
  }
});

// node_modules/unenv/dist/runtime/node/internal/process/node-version.mjs
var NODE_VERSION;
var init_node_version = __esm({
  "node_modules/unenv/dist/runtime/node/internal/process/node-version.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    NODE_VERSION = "22.14.0";
  }
});

// node_modules/unenv/dist/runtime/node/internal/process/process.mjs
import { EventEmitter } from "node:events";
var Process;
var init_process = __esm({
  "node_modules/unenv/dist/runtime/node/internal/process/process.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    init_tty();
    init_utils();
    init_node_version();
    Process = class _Process extends EventEmitter {
      static {
        __name(this, "Process");
      }
      env;
      hrtime;
      nextTick;
      constructor(impl) {
        super();
        this.env = impl.env;
        this.hrtime = impl.hrtime;
        this.nextTick = impl.nextTick;
        for (const prop of [...Object.getOwnPropertyNames(_Process.prototype), ...Object.getOwnPropertyNames(EventEmitter.prototype)]) {
          const value = this[prop];
          if (typeof value === "function") {
            this[prop] = value.bind(this);
          }
        }
      }
      // --- event emitter ---
      emitWarning(warning, type, code) {
        console.warn(`${code ? `[${code}] ` : ""}${type ? `${type}: ` : ""}${warning}`);
      }
      emit(...args) {
        return super.emit(...args);
      }
      listeners(eventName) {
        return super.listeners(eventName);
      }
      // --- stdio (lazy initializers) ---
      #stdin;
      #stdout;
      #stderr;
      get stdin() {
        return this.#stdin ??= new ReadStream(0);
      }
      get stdout() {
        return this.#stdout ??= new WriteStream(1);
      }
      get stderr() {
        return this.#stderr ??= new WriteStream(2);
      }
      // --- cwd ---
      #cwd = "/";
      chdir(cwd2) {
        this.#cwd = cwd2;
      }
      cwd() {
        return this.#cwd;
      }
      // --- dummy props and getters ---
      arch = "";
      platform = "";
      argv = [];
      argv0 = "";
      execArgv = [];
      execPath = "";
      title = "";
      pid = 200;
      ppid = 100;
      get version() {
        return `v${NODE_VERSION}`;
      }
      get versions() {
        return { node: NODE_VERSION };
      }
      get allowedNodeEnvironmentFlags() {
        return /* @__PURE__ */ new Set();
      }
      get sourceMapsEnabled() {
        return false;
      }
      get debugPort() {
        return 0;
      }
      get throwDeprecation() {
        return false;
      }
      get traceDeprecation() {
        return false;
      }
      get features() {
        return {};
      }
      get release() {
        return {};
      }
      get connected() {
        return false;
      }
      get config() {
        return {};
      }
      get moduleLoadList() {
        return [];
      }
      constrainedMemory() {
        return 0;
      }
      availableMemory() {
        return 0;
      }
      uptime() {
        return 0;
      }
      resourceUsage() {
        return {};
      }
      // --- noop methods ---
      ref() {
      }
      unref() {
      }
      // --- unimplemented methods ---
      umask() {
        throw createNotImplementedError("process.umask");
      }
      getBuiltinModule() {
        return void 0;
      }
      getActiveResourcesInfo() {
        throw createNotImplementedError("process.getActiveResourcesInfo");
      }
      exit() {
        throw createNotImplementedError("process.exit");
      }
      reallyExit() {
        throw createNotImplementedError("process.reallyExit");
      }
      kill() {
        throw createNotImplementedError("process.kill");
      }
      abort() {
        throw createNotImplementedError("process.abort");
      }
      dlopen() {
        throw createNotImplementedError("process.dlopen");
      }
      setSourceMapsEnabled() {
        throw createNotImplementedError("process.setSourceMapsEnabled");
      }
      loadEnvFile() {
        throw createNotImplementedError("process.loadEnvFile");
      }
      disconnect() {
        throw createNotImplementedError("process.disconnect");
      }
      cpuUsage() {
        throw createNotImplementedError("process.cpuUsage");
      }
      setUncaughtExceptionCaptureCallback() {
        throw createNotImplementedError("process.setUncaughtExceptionCaptureCallback");
      }
      hasUncaughtExceptionCaptureCallback() {
        throw createNotImplementedError("process.hasUncaughtExceptionCaptureCallback");
      }
      initgroups() {
        throw createNotImplementedError("process.initgroups");
      }
      openStdin() {
        throw createNotImplementedError("process.openStdin");
      }
      assert() {
        throw createNotImplementedError("process.assert");
      }
      binding() {
        throw createNotImplementedError("process.binding");
      }
      // --- attached interfaces ---
      permission = { has: /* @__PURE__ */ notImplemented("process.permission.has") };
      report = {
        directory: "",
        filename: "",
        signal: "SIGUSR2",
        compact: false,
        reportOnFatalError: false,
        reportOnSignal: false,
        reportOnUncaughtException: false,
        getReport: /* @__PURE__ */ notImplemented("process.report.getReport"),
        writeReport: /* @__PURE__ */ notImplemented("process.report.writeReport")
      };
      finalization = {
        register: /* @__PURE__ */ notImplemented("process.finalization.register"),
        unregister: /* @__PURE__ */ notImplemented("process.finalization.unregister"),
        registerBeforeExit: /* @__PURE__ */ notImplemented("process.finalization.registerBeforeExit")
      };
      memoryUsage = Object.assign(() => ({
        arrayBuffers: 0,
        rss: 0,
        external: 0,
        heapTotal: 0,
        heapUsed: 0
      }), { rss: /* @__PURE__ */ __name(() => 0, "rss") });
      // --- undefined props ---
      mainModule = void 0;
      domain = void 0;
      // optional
      send = void 0;
      exitCode = void 0;
      channel = void 0;
      getegid = void 0;
      geteuid = void 0;
      getgid = void 0;
      getgroups = void 0;
      getuid = void 0;
      setegid = void 0;
      seteuid = void 0;
      setgid = void 0;
      setgroups = void 0;
      setuid = void 0;
      // internals
      _events = void 0;
      _eventsCount = void 0;
      _exiting = void 0;
      _maxListeners = void 0;
      _debugEnd = void 0;
      _debugProcess = void 0;
      _fatalException = void 0;
      _getActiveHandles = void 0;
      _getActiveRequests = void 0;
      _kill = void 0;
      _preload_modules = void 0;
      _rawDebug = void 0;
      _startProfilerIdleNotifier = void 0;
      _stopProfilerIdleNotifier = void 0;
      _tickCallback = void 0;
      _disconnect = void 0;
      _handleQueue = void 0;
      _pendingMessage = void 0;
      _channel = void 0;
      _send = void 0;
      _linkedBinding = void 0;
    };
  }
});

// node_modules/@cloudflare/unenv-preset/dist/runtime/node/process.mjs
var globalProcess, getBuiltinModule, workerdProcess, isWorkerdProcessV2, unenvProcess, exit, features, platform, env, hrtime3, nextTick, _channel, _disconnect, _events, _eventsCount, _handleQueue, _maxListeners, _pendingMessage, _send, assert2, disconnect, mainModule, _debugEnd, _debugProcess, _exiting, _fatalException, _getActiveHandles, _getActiveRequests, _kill, _linkedBinding, _preload_modules, _rawDebug, _startProfilerIdleNotifier, _stopProfilerIdleNotifier, _tickCallback, abort, addListener, allowedNodeEnvironmentFlags, arch, argv, argv0, availableMemory, binding, channel, chdir, config, connected, constrainedMemory, cpuUsage, cwd, debugPort, dlopen, domain, emit, emitWarning, eventNames, execArgv, execPath, exitCode, finalization, getActiveResourcesInfo, getegid, geteuid, getgid, getgroups, getMaxListeners, getuid, hasUncaughtExceptionCaptureCallback, initgroups, kill, listenerCount, listeners, loadEnvFile, memoryUsage, moduleLoadList, off, on, once, openStdin, permission, pid, ppid, prependListener, prependOnceListener, rawListeners, reallyExit, ref, release, removeAllListeners, removeListener, report, resourceUsage, send, setegid, seteuid, setgid, setgroups, setMaxListeners, setSourceMapsEnabled, setuid, setUncaughtExceptionCaptureCallback, sourceMapsEnabled, stderr, stdin, stdout, throwDeprecation, title, traceDeprecation, umask, unref, uptime, version, versions, _process, process_default;
var init_process2 = __esm({
  "node_modules/@cloudflare/unenv-preset/dist/runtime/node/process.mjs"() {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    init_hrtime();
    init_process();
    globalProcess = globalThis["process"];
    getBuiltinModule = globalProcess.getBuiltinModule;
    workerdProcess = getBuiltinModule("node:process");
    isWorkerdProcessV2 = globalThis.Cloudflare.compatibilityFlags.enable_nodejs_process_v2;
    unenvProcess = new Process({
      env: globalProcess.env,
      // `hrtime` is only available from workerd process v2
      hrtime: isWorkerdProcessV2 ? workerdProcess.hrtime : hrtime,
      // `nextTick` is available from workerd process v1
      nextTick: workerdProcess.nextTick
    });
    ({ exit, features, platform } = workerdProcess);
    ({
      env: (
        // Always implemented by workerd
        env
      ),
      hrtime: (
        // Only implemented in workerd v2
        hrtime3
      ),
      nextTick: (
        // Always implemented by workerd
        nextTick
      )
    } = unenvProcess);
    ({
      _channel,
      _disconnect,
      _events,
      _eventsCount,
      _handleQueue,
      _maxListeners,
      _pendingMessage,
      _send,
      assert: assert2,
      disconnect,
      mainModule
    } = unenvProcess);
    ({
      _debugEnd: (
        // @ts-expect-error `_debugEnd` is missing typings
        _debugEnd
      ),
      _debugProcess: (
        // @ts-expect-error `_debugProcess` is missing typings
        _debugProcess
      ),
      _exiting: (
        // @ts-expect-error `_exiting` is missing typings
        _exiting
      ),
      _fatalException: (
        // @ts-expect-error `_fatalException` is missing typings
        _fatalException
      ),
      _getActiveHandles: (
        // @ts-expect-error `_getActiveHandles` is missing typings
        _getActiveHandles
      ),
      _getActiveRequests: (
        // @ts-expect-error `_getActiveRequests` is missing typings
        _getActiveRequests
      ),
      _kill: (
        // @ts-expect-error `_kill` is missing typings
        _kill
      ),
      _linkedBinding: (
        // @ts-expect-error `_linkedBinding` is missing typings
        _linkedBinding
      ),
      _preload_modules: (
        // @ts-expect-error `_preload_modules` is missing typings
        _preload_modules
      ),
      _rawDebug: (
        // @ts-expect-error `_rawDebug` is missing typings
        _rawDebug
      ),
      _startProfilerIdleNotifier: (
        // @ts-expect-error `_startProfilerIdleNotifier` is missing typings
        _startProfilerIdleNotifier
      ),
      _stopProfilerIdleNotifier: (
        // @ts-expect-error `_stopProfilerIdleNotifier` is missing typings
        _stopProfilerIdleNotifier
      ),
      _tickCallback: (
        // @ts-expect-error `_tickCallback` is missing typings
        _tickCallback
      ),
      abort,
      addListener,
      allowedNodeEnvironmentFlags,
      arch,
      argv,
      argv0,
      availableMemory,
      binding: (
        // @ts-expect-error `binding` is missing typings
        binding
      ),
      channel,
      chdir,
      config,
      connected,
      constrainedMemory,
      cpuUsage,
      cwd,
      debugPort,
      dlopen,
      domain: (
        // @ts-expect-error `domain` is missing typings
        domain
      ),
      emit,
      emitWarning,
      eventNames,
      execArgv,
      execPath,
      exitCode,
      finalization,
      getActiveResourcesInfo,
      getegid,
      geteuid,
      getgid,
      getgroups,
      getMaxListeners,
      getuid,
      hasUncaughtExceptionCaptureCallback,
      initgroups: (
        // @ts-expect-error `initgroups` is missing typings
        initgroups
      ),
      kill,
      listenerCount,
      listeners,
      loadEnvFile,
      memoryUsage,
      moduleLoadList: (
        // @ts-expect-error `moduleLoadList` is missing typings
        moduleLoadList
      ),
      off,
      on,
      once,
      openStdin: (
        // @ts-expect-error `openStdin` is missing typings
        openStdin
      ),
      permission,
      pid,
      ppid,
      prependListener,
      prependOnceListener,
      rawListeners,
      reallyExit: (
        // @ts-expect-error `reallyExit` is missing typings
        reallyExit
      ),
      ref,
      release,
      removeAllListeners,
      removeListener,
      report,
      resourceUsage,
      send,
      setegid,
      seteuid,
      setgid,
      setgroups,
      setMaxListeners,
      setSourceMapsEnabled,
      setuid,
      setUncaughtExceptionCaptureCallback,
      sourceMapsEnabled,
      stderr,
      stdin,
      stdout,
      throwDeprecation,
      title,
      traceDeprecation,
      umask,
      unref,
      uptime,
      version,
      versions
    } = isWorkerdProcessV2 ? workerdProcess : unenvProcess);
    _process = {
      abort,
      addListener,
      allowedNodeEnvironmentFlags,
      hasUncaughtExceptionCaptureCallback,
      setUncaughtExceptionCaptureCallback,
      loadEnvFile,
      sourceMapsEnabled,
      arch,
      argv,
      argv0,
      chdir,
      config,
      connected,
      constrainedMemory,
      availableMemory,
      cpuUsage,
      cwd,
      debugPort,
      dlopen,
      disconnect,
      emit,
      emitWarning,
      env,
      eventNames,
      execArgv,
      execPath,
      exit,
      finalization,
      features,
      getBuiltinModule,
      getActiveResourcesInfo,
      getMaxListeners,
      hrtime: hrtime3,
      kill,
      listeners,
      listenerCount,
      memoryUsage,
      nextTick,
      on,
      off,
      once,
      pid,
      platform,
      ppid,
      prependListener,
      prependOnceListener,
      rawListeners,
      release,
      removeAllListeners,
      removeListener,
      report,
      resourceUsage,
      setMaxListeners,
      setSourceMapsEnabled,
      stderr,
      stdin,
      stdout,
      title,
      throwDeprecation,
      traceDeprecation,
      umask,
      uptime,
      version,
      versions,
      // @ts-expect-error old API
      domain,
      initgroups,
      moduleLoadList,
      reallyExit,
      openStdin,
      assert: assert2,
      binding,
      send,
      exitCode,
      channel,
      getegid,
      geteuid,
      getgid,
      getgroups,
      getuid,
      setegid,
      seteuid,
      setgid,
      setgroups,
      setuid,
      permission,
      mainModule,
      _events,
      _eventsCount,
      _exiting,
      _maxListeners,
      _debugEnd,
      _debugProcess,
      _fatalException,
      _getActiveHandles,
      _getActiveRequests,
      _kill,
      _preload_modules,
      _rawDebug,
      _startProfilerIdleNotifier,
      _stopProfilerIdleNotifier,
      _tickCallback,
      _disconnect,
      _handleQueue,
      _pendingMessage,
      _channel,
      _send,
      _linkedBinding
    };
    process_default = _process;
  }
});

// node_modules/wrangler/_virtual_unenv_global_polyfill-@cloudflare-unenv-preset-node-process
var init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process = __esm({
  "node_modules/wrangler/_virtual_unenv_global_polyfill-@cloudflare-unenv-preset-node-process"() {
    init_process2();
    globalThis.process = process_default;
  }
});

// node_modules/bignumber.js/bignumber.js
var require_bignumber = __commonJS({
  "node_modules/bignumber.js/bignumber.js"(exports, module2) {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    (function(globalObject) {
      "use strict";
      var BigNumber, isNumeric = /^-?(?:\d+(?:\.\d*)?|\.\d+)(?:e[+-]?\d+)?$/i, mathceil = Math.ceil, mathfloor = Math.floor, bignumberError = "[BigNumber Error] ", tooManyDigits = bignumberError + "Number primitive has more than 15 significant digits: ", BASE = 1e14, LOG_BASE = 14, MAX_SAFE_INTEGER = 9007199254740991, POWS_TEN = [1, 10, 100, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11, 1e12, 1e13], SQRT_BASE = 1e7, MAX = 1e9;
      function clone(configObject) {
        var div, convertBase, parseNumeric, P = BigNumber2.prototype = { constructor: BigNumber2, toString: null, valueOf: null }, ONE = new BigNumber2(1), DECIMAL_PLACES = 20, ROUNDING_MODE = 4, TO_EXP_NEG = -7, TO_EXP_POS = 21, MIN_EXP = -1e7, MAX_EXP = 1e7, CRYPTO = false, MODULO_MODE = 1, POW_PRECISION = 0, FORMAT = {
          prefix: "",
          groupSize: 3,
          secondaryGroupSize: 0,
          groupSeparator: ",",
          decimalSeparator: ".",
          fractionGroupSize: 0,
          fractionGroupSeparator: "\xA0",
          // non-breaking space
          suffix: ""
        }, ALPHABET = "0123456789abcdefghijklmnopqrstuvwxyz", alphabetHasNormalDecimalDigits = true;
        function BigNumber2(v, b) {
          var alphabet, c, caseChanged, e3, i3, isNum, len, str, x = this;
          if (!(x instanceof BigNumber2)) return new BigNumber2(v, b);
          if (b == null) {
            if (v && v._isBigNumber === true) {
              x.s = v.s;
              if (!v.c || v.e > MAX_EXP) {
                x.c = x.e = null;
              } else if (v.e < MIN_EXP) {
                x.c = [x.e = 0];
              } else {
                x.e = v.e;
                x.c = v.c.slice();
              }
              return;
            }
            if ((isNum = typeof v == "number") && v * 0 == 0) {
              x.s = 1 / v < 0 ? (v = -v, -1) : 1;
              if (v === ~~v) {
                for (e3 = 0, i3 = v; i3 >= 10; i3 /= 10, e3++) ;
                if (e3 > MAX_EXP) {
                  x.c = x.e = null;
                } else {
                  x.e = e3;
                  x.c = [v];
                }
                return;
              }
              str = String(v);
            } else {
              if (!isNumeric.test(str = String(v))) return parseNumeric(x, str, isNum);
              x.s = str.charCodeAt(0) == 45 ? (str = str.slice(1), -1) : 1;
            }
            if ((e3 = str.indexOf(".")) > -1) str = str.replace(".", "");
            if ((i3 = str.search(/e/i)) > 0) {
              if (e3 < 0) e3 = i3;
              e3 += +str.slice(i3 + 1);
              str = str.substring(0, i3);
            } else if (e3 < 0) {
              e3 = str.length;
            }
          } else {
            intCheck(b, 2, ALPHABET.length, "Base");
            if (b == 10 && alphabetHasNormalDecimalDigits) {
              x = new BigNumber2(v);
              return round(x, DECIMAL_PLACES + x.e + 1, ROUNDING_MODE);
            }
            str = String(v);
            if (isNum = typeof v == "number") {
              if (v * 0 != 0) return parseNumeric(x, str, isNum, b);
              x.s = 1 / v < 0 ? (str = str.slice(1), -1) : 1;
              if (BigNumber2.DEBUG && str.replace(/^0\.0*|\./, "").length > 15) {
                throw Error(tooManyDigits + v);
              }
            } else {
              x.s = str.charCodeAt(0) === 45 ? (str = str.slice(1), -1) : 1;
            }
            alphabet = ALPHABET.slice(0, b);
            e3 = i3 = 0;
            for (len = str.length; i3 < len; i3++) {
              if (alphabet.indexOf(c = str.charAt(i3)) < 0) {
                if (c == ".") {
                  if (i3 > e3) {
                    e3 = len;
                    continue;
                  }
                } else if (!caseChanged) {
                  if (str == str.toUpperCase() && (str = str.toLowerCase()) || str == str.toLowerCase() && (str = str.toUpperCase())) {
                    caseChanged = true;
                    i3 = -1;
                    e3 = 0;
                    continue;
                  }
                }
                return parseNumeric(x, String(v), isNum, b);
              }
            }
            isNum = false;
            str = convertBase(str, b, 10, x.s);
            if ((e3 = str.indexOf(".")) > -1) str = str.replace(".", "");
            else e3 = str.length;
          }
          for (i3 = 0; str.charCodeAt(i3) === 48; i3++) ;
          for (len = str.length; str.charCodeAt(--len) === 48; ) ;
          if (str = str.slice(i3, ++len)) {
            len -= i3;
            if (isNum && BigNumber2.DEBUG && len > 15 && (v > MAX_SAFE_INTEGER || v !== mathfloor(v))) {
              throw Error(tooManyDigits + x.s * v);
            }
            if ((e3 = e3 - i3 - 1) > MAX_EXP) {
              x.c = x.e = null;
            } else if (e3 < MIN_EXP) {
              x.c = [x.e = 0];
            } else {
              x.e = e3;
              x.c = [];
              i3 = (e3 + 1) % LOG_BASE;
              if (e3 < 0) i3 += LOG_BASE;
              if (i3 < len) {
                if (i3) x.c.push(+str.slice(0, i3));
                for (len -= LOG_BASE; i3 < len; ) {
                  x.c.push(+str.slice(i3, i3 += LOG_BASE));
                }
                i3 = LOG_BASE - (str = str.slice(i3)).length;
              } else {
                i3 -= len;
              }
              for (; i3--; str += "0") ;
              x.c.push(+str);
            }
          } else {
            x.c = [x.e = 0];
          }
        }
        __name(BigNumber2, "BigNumber");
        BigNumber2.clone = clone;
        BigNumber2.ROUND_UP = 0;
        BigNumber2.ROUND_DOWN = 1;
        BigNumber2.ROUND_CEIL = 2;
        BigNumber2.ROUND_FLOOR = 3;
        BigNumber2.ROUND_HALF_UP = 4;
        BigNumber2.ROUND_HALF_DOWN = 5;
        BigNumber2.ROUND_HALF_EVEN = 6;
        BigNumber2.ROUND_HALF_CEIL = 7;
        BigNumber2.ROUND_HALF_FLOOR = 8;
        BigNumber2.EUCLID = 9;
        BigNumber2.config = BigNumber2.set = function(obj) {
          var p, v;
          if (obj != null) {
            if (typeof obj == "object") {
              if (obj.hasOwnProperty(p = "DECIMAL_PLACES")) {
                v = obj[p];
                intCheck(v, 0, MAX, p);
                DECIMAL_PLACES = v;
              }
              if (obj.hasOwnProperty(p = "ROUNDING_MODE")) {
                v = obj[p];
                intCheck(v, 0, 8, p);
                ROUNDING_MODE = v;
              }
              if (obj.hasOwnProperty(p = "EXPONENTIAL_AT")) {
                v = obj[p];
                if (v && v.pop) {
                  intCheck(v[0], -MAX, 0, p);
                  intCheck(v[1], 0, MAX, p);
                  TO_EXP_NEG = v[0];
                  TO_EXP_POS = v[1];
                } else {
                  intCheck(v, -MAX, MAX, p);
                  TO_EXP_NEG = -(TO_EXP_POS = v < 0 ? -v : v);
                }
              }
              if (obj.hasOwnProperty(p = "RANGE")) {
                v = obj[p];
                if (v && v.pop) {
                  intCheck(v[0], -MAX, -1, p);
                  intCheck(v[1], 1, MAX, p);
                  MIN_EXP = v[0];
                  MAX_EXP = v[1];
                } else {
                  intCheck(v, -MAX, MAX, p);
                  if (v) {
                    MIN_EXP = -(MAX_EXP = v < 0 ? -v : v);
                  } else {
                    throw Error(bignumberError + p + " cannot be zero: " + v);
                  }
                }
              }
              if (obj.hasOwnProperty(p = "CRYPTO")) {
                v = obj[p];
                if (v === !!v) {
                  if (v) {
                    if (typeof crypto != "undefined" && crypto && (crypto.getRandomValues || crypto.randomBytes)) {
                      CRYPTO = v;
                    } else {
                      CRYPTO = !v;
                      throw Error(bignumberError + "crypto unavailable");
                    }
                  } else {
                    CRYPTO = v;
                  }
                } else {
                  throw Error(bignumberError + p + " not true or false: " + v);
                }
              }
              if (obj.hasOwnProperty(p = "MODULO_MODE")) {
                v = obj[p];
                intCheck(v, 0, 9, p);
                MODULO_MODE = v;
              }
              if (obj.hasOwnProperty(p = "POW_PRECISION")) {
                v = obj[p];
                intCheck(v, 0, MAX, p);
                POW_PRECISION = v;
              }
              if (obj.hasOwnProperty(p = "FORMAT")) {
                v = obj[p];
                if (typeof v == "object") FORMAT = v;
                else throw Error(bignumberError + p + " not an object: " + v);
              }
              if (obj.hasOwnProperty(p = "ALPHABET")) {
                v = obj[p];
                if (typeof v == "string" && !/^.?$|[+\-.\s]|(.).*\1/.test(v)) {
                  alphabetHasNormalDecimalDigits = v.slice(0, 10) == "0123456789";
                  ALPHABET = v;
                } else {
                  throw Error(bignumberError + p + " invalid: " + v);
                }
              }
            } else {
              throw Error(bignumberError + "Object expected: " + obj);
            }
          }
          return {
            DECIMAL_PLACES,
            ROUNDING_MODE,
            EXPONENTIAL_AT: [TO_EXP_NEG, TO_EXP_POS],
            RANGE: [MIN_EXP, MAX_EXP],
            CRYPTO,
            MODULO_MODE,
            POW_PRECISION,
            FORMAT,
            ALPHABET
          };
        };
        BigNumber2.isBigNumber = function(v) {
          if (!v || v._isBigNumber !== true) return false;
          if (!BigNumber2.DEBUG) return true;
          var i3, n2, c = v.c, e3 = v.e, s2 = v.s;
          out: if ({}.toString.call(c) == "[object Array]") {
            if ((s2 === 1 || s2 === -1) && e3 >= -MAX && e3 <= MAX && e3 === mathfloor(e3)) {
              if (c[0] === 0) {
                if (e3 === 0 && c.length === 1) return true;
                break out;
              }
              i3 = (e3 + 1) % LOG_BASE;
              if (i3 < 1) i3 += LOG_BASE;
              if (String(c[0]).length == i3) {
                for (i3 = 0; i3 < c.length; i3++) {
                  n2 = c[i3];
                  if (n2 < 0 || n2 >= BASE || n2 !== mathfloor(n2)) break out;
                }
                if (n2 !== 0) return true;
              }
            }
          } else if (c === null && e3 === null && (s2 === null || s2 === 1 || s2 === -1)) {
            return true;
          }
          throw Error(bignumberError + "Invalid BigNumber: " + v);
        };
        BigNumber2.maximum = BigNumber2.max = function() {
          return maxOrMin(arguments, -1);
        };
        BigNumber2.minimum = BigNumber2.min = function() {
          return maxOrMin(arguments, 1);
        };
        BigNumber2.random = (function() {
          var pow2_53 = 9007199254740992;
          var random53bitInt = Math.random() * pow2_53 & 2097151 ? function() {
            return mathfloor(Math.random() * pow2_53);
          } : function() {
            return (Math.random() * 1073741824 | 0) * 8388608 + (Math.random() * 8388608 | 0);
          };
          return function(dp) {
            var a3, b, e3, k, v, i3 = 0, c = [], rand = new BigNumber2(ONE);
            if (dp == null) dp = DECIMAL_PLACES;
            else intCheck(dp, 0, MAX);
            k = mathceil(dp / LOG_BASE);
            if (CRYPTO) {
              if (crypto.getRandomValues) {
                a3 = crypto.getRandomValues(new Uint32Array(k *= 2));
                for (; i3 < k; ) {
                  v = a3[i3] * 131072 + (a3[i3 + 1] >>> 11);
                  if (v >= 9e15) {
                    b = crypto.getRandomValues(new Uint32Array(2));
                    a3[i3] = b[0];
                    a3[i3 + 1] = b[1];
                  } else {
                    c.push(v % 1e14);
                    i3 += 2;
                  }
                }
                i3 = k / 2;
              } else if (crypto.randomBytes) {
                a3 = crypto.randomBytes(k *= 7);
                for (; i3 < k; ) {
                  v = (a3[i3] & 31) * 281474976710656 + a3[i3 + 1] * 1099511627776 + a3[i3 + 2] * 4294967296 + a3[i3 + 3] * 16777216 + (a3[i3 + 4] << 16) + (a3[i3 + 5] << 8) + a3[i3 + 6];
                  if (v >= 9e15) {
                    crypto.randomBytes(7).copy(a3, i3);
                  } else {
                    c.push(v % 1e14);
                    i3 += 7;
                  }
                }
                i3 = k / 7;
              } else {
                CRYPTO = false;
                throw Error(bignumberError + "crypto unavailable");
              }
            }
            if (!CRYPTO) {
              for (; i3 < k; ) {
                v = random53bitInt();
                if (v < 9e15) c[i3++] = v % 1e14;
              }
            }
            k = c[--i3];
            dp %= LOG_BASE;
            if (k && dp) {
              v = POWS_TEN[LOG_BASE - dp];
              c[i3] = mathfloor(k / v) * v;
            }
            for (; c[i3] === 0; c.pop(), i3--) ;
            if (i3 < 0) {
              c = [e3 = 0];
            } else {
              for (e3 = -1; c[0] === 0; c.splice(0, 1), e3 -= LOG_BASE) ;
              for (i3 = 1, v = c[0]; v >= 10; v /= 10, i3++) ;
              if (i3 < LOG_BASE) e3 -= LOG_BASE - i3;
            }
            rand.e = e3;
            rand.c = c;
            return rand;
          };
        })();
        BigNumber2.sum = function() {
          var i3 = 1, args = arguments, sum = new BigNumber2(args[0]);
          for (; i3 < args.length; ) sum = sum.plus(args[i3++]);
          return sum;
        };
        convertBase = /* @__PURE__ */ (function() {
          var decimal = "0123456789";
          function toBaseOut(str, baseIn, baseOut, alphabet) {
            var j, arr = [0], arrL, i3 = 0, len = str.length;
            for (; i3 < len; ) {
              for (arrL = arr.length; arrL--; arr[arrL] *= baseIn) ;
              arr[0] += alphabet.indexOf(str.charAt(i3++));
              for (j = 0; j < arr.length; j++) {
                if (arr[j] > baseOut - 1) {
                  if (arr[j + 1] == null) arr[j + 1] = 0;
                  arr[j + 1] += arr[j] / baseOut | 0;
                  arr[j] %= baseOut;
                }
              }
            }
            return arr.reverse();
          }
          __name(toBaseOut, "toBaseOut");
          return function(str, baseIn, baseOut, sign, callerIsToString) {
            var alphabet, d, e3, k, r2, x, xc, y, i3 = str.indexOf("."), dp = DECIMAL_PLACES, rm = ROUNDING_MODE;
            if (i3 >= 0) {
              k = POW_PRECISION;
              POW_PRECISION = 0;
              str = str.replace(".", "");
              y = new BigNumber2(baseIn);
              x = y.pow(str.length - i3);
              POW_PRECISION = k;
              y.c = toBaseOut(
                toFixedPoint(coeffToString(x.c), x.e, "0"),
                10,
                baseOut,
                decimal
              );
              y.e = y.c.length;
            }
            xc = toBaseOut(str, baseIn, baseOut, callerIsToString ? (alphabet = ALPHABET, decimal) : (alphabet = decimal, ALPHABET));
            e3 = k = xc.length;
            for (; xc[--k] == 0; xc.pop()) ;
            if (!xc[0]) return alphabet.charAt(0);
            if (i3 < 0) {
              --e3;
            } else {
              x.c = xc;
              x.e = e3;
              x.s = sign;
              x = div(x, y, dp, rm, baseOut);
              xc = x.c;
              r2 = x.r;
              e3 = x.e;
            }
            d = e3 + dp + 1;
            i3 = xc[d];
            k = baseOut / 2;
            r2 = r2 || d < 0 || xc[d + 1] != null;
            r2 = rm < 4 ? (i3 != null || r2) && (rm == 0 || rm == (x.s < 0 ? 3 : 2)) : i3 > k || i3 == k && (rm == 4 || r2 || rm == 6 && xc[d - 1] & 1 || rm == (x.s < 0 ? 8 : 7));
            if (d < 1 || !xc[0]) {
              str = r2 ? toFixedPoint(alphabet.charAt(1), -dp, alphabet.charAt(0)) : alphabet.charAt(0);
            } else {
              xc.length = d;
              if (r2) {
                for (--baseOut; ++xc[--d] > baseOut; ) {
                  xc[d] = 0;
                  if (!d) {
                    ++e3;
                    xc = [1].concat(xc);
                  }
                }
              }
              for (k = xc.length; !xc[--k]; ) ;
              for (i3 = 0, str = ""; i3 <= k; str += alphabet.charAt(xc[i3++])) ;
              str = toFixedPoint(str, e3, alphabet.charAt(0));
            }
            return str;
          };
        })();
        div = /* @__PURE__ */ (function() {
          function multiply(x, k, base) {
            var m, temp, xlo, xhi, carry = 0, i3 = x.length, klo = k % SQRT_BASE, khi = k / SQRT_BASE | 0;
            for (x = x.slice(); i3--; ) {
              xlo = x[i3] % SQRT_BASE;
              xhi = x[i3] / SQRT_BASE | 0;
              m = khi * xlo + xhi * klo;
              temp = klo * xlo + m % SQRT_BASE * SQRT_BASE + carry;
              carry = (temp / base | 0) + (m / SQRT_BASE | 0) + khi * xhi;
              x[i3] = temp % base;
            }
            if (carry) x = [carry].concat(x);
            return x;
          }
          __name(multiply, "multiply");
          function compare2(a3, b, aL, bL) {
            var i3, cmp;
            if (aL != bL) {
              cmp = aL > bL ? 1 : -1;
            } else {
              for (i3 = cmp = 0; i3 < aL; i3++) {
                if (a3[i3] != b[i3]) {
                  cmp = a3[i3] > b[i3] ? 1 : -1;
                  break;
                }
              }
            }
            return cmp;
          }
          __name(compare2, "compare");
          function subtract(a3, b, aL, base) {
            var i3 = 0;
            for (; aL--; ) {
              a3[aL] -= i3;
              i3 = a3[aL] < b[aL] ? 1 : 0;
              a3[aL] = i3 * base + a3[aL] - b[aL];
            }
            for (; !a3[0] && a3.length > 1; a3.splice(0, 1)) ;
          }
          __name(subtract, "subtract");
          return function(x, y, dp, rm, base) {
            var cmp, e3, i3, more, n2, prod, prodL, q, qc, rem, remL, rem0, xi, xL, yc0, yL, yz, s2 = x.s == y.s ? 1 : -1, xc = x.c, yc = y.c;
            if (!xc || !xc[0] || !yc || !yc[0]) {
              return new BigNumber2(
                // Return NaN if either NaN, or both Infinity or 0.
                !x.s || !y.s || (xc ? yc && xc[0] == yc[0] : !yc) ? NaN : (
                  // Return ±0 if x is ±0 or y is ±Infinity, or return ±Infinity as y is ±0.
                  xc && xc[0] == 0 || !yc ? s2 * 0 : s2 / 0
                )
              );
            }
            q = new BigNumber2(s2);
            qc = q.c = [];
            e3 = x.e - y.e;
            s2 = dp + e3 + 1;
            if (!base) {
              base = BASE;
              e3 = bitFloor(x.e / LOG_BASE) - bitFloor(y.e / LOG_BASE);
              s2 = s2 / LOG_BASE | 0;
            }
            for (i3 = 0; yc[i3] == (xc[i3] || 0); i3++) ;
            if (yc[i3] > (xc[i3] || 0)) e3--;
            if (s2 < 0) {
              qc.push(1);
              more = true;
            } else {
              xL = xc.length;
              yL = yc.length;
              i3 = 0;
              s2 += 2;
              n2 = mathfloor(base / (yc[0] + 1));
              if (n2 > 1) {
                yc = multiply(yc, n2, base);
                xc = multiply(xc, n2, base);
                yL = yc.length;
                xL = xc.length;
              }
              xi = yL;
              rem = xc.slice(0, yL);
              remL = rem.length;
              for (; remL < yL; rem[remL++] = 0) ;
              yz = yc.slice();
              yz = [0].concat(yz);
              yc0 = yc[0];
              if (yc[1] >= base / 2) yc0++;
              do {
                n2 = 0;
                cmp = compare2(yc, rem, yL, remL);
                if (cmp < 0) {
                  rem0 = rem[0];
                  if (yL != remL) rem0 = rem0 * base + (rem[1] || 0);
                  n2 = mathfloor(rem0 / yc0);
                  if (n2 > 1) {
                    if (n2 >= base) n2 = base - 1;
                    prod = multiply(yc, n2, base);
                    prodL = prod.length;
                    remL = rem.length;
                    while (compare2(prod, rem, prodL, remL) == 1) {
                      n2--;
                      subtract(prod, yL < prodL ? yz : yc, prodL, base);
                      prodL = prod.length;
                      cmp = 1;
                    }
                  } else {
                    if (n2 == 0) {
                      cmp = n2 = 1;
                    }
                    prod = yc.slice();
                    prodL = prod.length;
                  }
                  if (prodL < remL) prod = [0].concat(prod);
                  subtract(rem, prod, remL, base);
                  remL = rem.length;
                  if (cmp == -1) {
                    while (compare2(yc, rem, yL, remL) < 1) {
                      n2++;
                      subtract(rem, yL < remL ? yz : yc, remL, base);
                      remL = rem.length;
                    }
                  }
                } else if (cmp === 0) {
                  n2++;
                  rem = [0];
                }
                qc[i3++] = n2;
                if (rem[0]) {
                  rem[remL++] = xc[xi] || 0;
                } else {
                  rem = [xc[xi]];
                  remL = 1;
                }
              } while ((xi++ < xL || rem[0] != null) && s2--);
              more = rem[0] != null;
              if (!qc[0]) qc.splice(0, 1);
            }
            if (base == BASE) {
              for (i3 = 1, s2 = qc[0]; s2 >= 10; s2 /= 10, i3++) ;
              round(q, dp + (q.e = i3 + e3 * LOG_BASE - 1) + 1, rm, more);
            } else {
              q.e = e3;
              q.r = +more;
            }
            return q;
          };
        })();
        function format(n2, i3, rm, id) {
          var c0, e3, ne, len, str;
          if (rm == null) rm = ROUNDING_MODE;
          else intCheck(rm, 0, 8);
          if (!n2.c) return n2.toString();
          c0 = n2.c[0];
          ne = n2.e;
          if (i3 == null) {
            str = coeffToString(n2.c);
            str = id == 1 || id == 2 && (ne <= TO_EXP_NEG || ne >= TO_EXP_POS) ? toExponential(str, ne) : toFixedPoint(str, ne, "0");
          } else {
            n2 = round(new BigNumber2(n2), i3, rm);
            e3 = n2.e;
            str = coeffToString(n2.c);
            len = str.length;
            if (id == 1 || id == 2 && (i3 <= e3 || e3 <= TO_EXP_NEG)) {
              for (; len < i3; str += "0", len++) ;
              str = toExponential(str, e3);
            } else {
              i3 -= ne + (id === 2 && e3 > ne);
              str = toFixedPoint(str, e3, "0");
              if (e3 + 1 > len) {
                if (--i3 > 0) for (str += "."; i3--; str += "0") ;
              } else {
                i3 += e3 - len;
                if (i3 > 0) {
                  if (e3 + 1 == len) str += ".";
                  for (; i3--; str += "0") ;
                }
              }
            }
          }
          return n2.s < 0 && c0 ? "-" + str : str;
        }
        __name(format, "format");
        function maxOrMin(args, n2) {
          var k, y, i3 = 1, x = new BigNumber2(args[0]);
          for (; i3 < args.length; i3++) {
            y = new BigNumber2(args[i3]);
            if (!y.s || (k = compare(x, y)) === n2 || k === 0 && x.s === n2) {
              x = y;
            }
          }
          return x;
        }
        __name(maxOrMin, "maxOrMin");
        function normalise(n2, c, e3) {
          var i3 = 1, j = c.length;
          for (; !c[--j]; c.pop()) ;
          for (j = c[0]; j >= 10; j /= 10, i3++) ;
          if ((e3 = i3 + e3 * LOG_BASE - 1) > MAX_EXP) {
            n2.c = n2.e = null;
          } else if (e3 < MIN_EXP) {
            n2.c = [n2.e = 0];
          } else {
            n2.e = e3;
            n2.c = c;
          }
          return n2;
        }
        __name(normalise, "normalise");
        parseNumeric = /* @__PURE__ */ (function() {
          var basePrefix = /^(-?)0([xbo])(?=\w[\w.]*$)/i, dotAfter = /^([^.]+)\.$/, dotBefore = /^\.([^.]+)$/, isInfinityOrNaN = /^-?(Infinity|NaN)$/, whitespaceOrPlus = /^\s*\+(?=[\w.])|^\s+|\s+$/g;
          return function(x, str, isNum, b) {
            var base, s2 = isNum ? str : str.replace(whitespaceOrPlus, "");
            if (isInfinityOrNaN.test(s2)) {
              x.s = isNaN(s2) ? null : s2 < 0 ? -1 : 1;
            } else {
              if (!isNum) {
                s2 = s2.replace(basePrefix, function(m, p1, p2) {
                  base = (p2 = p2.toLowerCase()) == "x" ? 16 : p2 == "b" ? 2 : 8;
                  return !b || b == base ? p1 : m;
                });
                if (b) {
                  base = b;
                  s2 = s2.replace(dotAfter, "$1").replace(dotBefore, "0.$1");
                }
                if (str != s2) return new BigNumber2(s2, base);
              }
              if (BigNumber2.DEBUG) {
                throw Error(bignumberError + "Not a" + (b ? " base " + b : "") + " number: " + str);
              }
              x.s = null;
            }
            x.c = x.e = null;
          };
        })();
        function round(x, sd, rm, r2) {
          var d, i3, j, k, n2, ni, rd, xc = x.c, pows10 = POWS_TEN;
          if (xc) {
            out: {
              for (d = 1, k = xc[0]; k >= 10; k /= 10, d++) ;
              i3 = sd - d;
              if (i3 < 0) {
                i3 += LOG_BASE;
                j = sd;
                n2 = xc[ni = 0];
                rd = mathfloor(n2 / pows10[d - j - 1] % 10);
              } else {
                ni = mathceil((i3 + 1) / LOG_BASE);
                if (ni >= xc.length) {
                  if (r2) {
                    for (; xc.length <= ni; xc.push(0)) ;
                    n2 = rd = 0;
                    d = 1;
                    i3 %= LOG_BASE;
                    j = i3 - LOG_BASE + 1;
                  } else {
                    break out;
                  }
                } else {
                  n2 = k = xc[ni];
                  for (d = 1; k >= 10; k /= 10, d++) ;
                  i3 %= LOG_BASE;
                  j = i3 - LOG_BASE + d;
                  rd = j < 0 ? 0 : mathfloor(n2 / pows10[d - j - 1] % 10);
                }
              }
              r2 = r2 || sd < 0 || // Are there any non-zero digits after the rounding digit?
              // The expression  n % pows10[d - j - 1]  returns all digits of n to the right
              // of the digit at j, e.g. if n is 908714 and j is 2, the expression gives 714.
              xc[ni + 1] != null || (j < 0 ? n2 : n2 % pows10[d - j - 1]);
              r2 = rm < 4 ? (rd || r2) && (rm == 0 || rm == (x.s < 0 ? 3 : 2)) : rd > 5 || rd == 5 && (rm == 4 || r2 || rm == 6 && // Check whether the digit to the left of the rounding digit is odd.
              (i3 > 0 ? j > 0 ? n2 / pows10[d - j] : 0 : xc[ni - 1]) % 10 & 1 || rm == (x.s < 0 ? 8 : 7));
              if (sd < 1 || !xc[0]) {
                xc.length = 0;
                if (r2) {
                  sd -= x.e + 1;
                  xc[0] = pows10[(LOG_BASE - sd % LOG_BASE) % LOG_BASE];
                  x.e = -sd || 0;
                } else {
                  xc[0] = x.e = 0;
                }
                return x;
              }
              if (i3 == 0) {
                xc.length = ni;
                k = 1;
                ni--;
              } else {
                xc.length = ni + 1;
                k = pows10[LOG_BASE - i3];
                xc[ni] = j > 0 ? mathfloor(n2 / pows10[d - j] % pows10[j]) * k : 0;
              }
              if (r2) {
                for (; ; ) {
                  if (ni == 0) {
                    for (i3 = 1, j = xc[0]; j >= 10; j /= 10, i3++) ;
                    j = xc[0] += k;
                    for (k = 1; j >= 10; j /= 10, k++) ;
                    if (i3 != k) {
                      x.e++;
                      if (xc[0] == BASE) xc[0] = 1;
                    }
                    break;
                  } else {
                    xc[ni] += k;
                    if (xc[ni] != BASE) break;
                    xc[ni--] = 0;
                    k = 1;
                  }
                }
              }
              for (i3 = xc.length; xc[--i3] === 0; xc.pop()) ;
            }
            if (x.e > MAX_EXP) {
              x.c = x.e = null;
            } else if (x.e < MIN_EXP) {
              x.c = [x.e = 0];
            }
          }
          return x;
        }
        __name(round, "round");
        function valueOf(n2) {
          var str, e3 = n2.e;
          if (e3 === null) return n2.toString();
          str = coeffToString(n2.c);
          str = e3 <= TO_EXP_NEG || e3 >= TO_EXP_POS ? toExponential(str, e3) : toFixedPoint(str, e3, "0");
          return n2.s < 0 ? "-" + str : str;
        }
        __name(valueOf, "valueOf");
        P.absoluteValue = P.abs = function() {
          var x = new BigNumber2(this);
          if (x.s < 0) x.s = 1;
          return x;
        };
        P.comparedTo = function(y, b) {
          return compare(this, new BigNumber2(y, b));
        };
        P.decimalPlaces = P.dp = function(dp, rm) {
          var c, n2, v, x = this;
          if (dp != null) {
            intCheck(dp, 0, MAX);
            if (rm == null) rm = ROUNDING_MODE;
            else intCheck(rm, 0, 8);
            return round(new BigNumber2(x), dp + x.e + 1, rm);
          }
          if (!(c = x.c)) return null;
          n2 = ((v = c.length - 1) - bitFloor(this.e / LOG_BASE)) * LOG_BASE;
          if (v = c[v]) for (; v % 10 == 0; v /= 10, n2--) ;
          if (n2 < 0) n2 = 0;
          return n2;
        };
        P.dividedBy = P.div = function(y, b) {
          return div(this, new BigNumber2(y, b), DECIMAL_PLACES, ROUNDING_MODE);
        };
        P.dividedToIntegerBy = P.idiv = function(y, b) {
          return div(this, new BigNumber2(y, b), 0, 1);
        };
        P.exponentiatedBy = P.pow = function(n2, m) {
          var half, isModExp, i3, k, more, nIsBig, nIsNeg, nIsOdd, y, x = this;
          n2 = new BigNumber2(n2);
          if (n2.c && !n2.isInteger()) {
            throw Error(bignumberError + "Exponent not an integer: " + valueOf(n2));
          }
          if (m != null) m = new BigNumber2(m);
          nIsBig = n2.e > 14;
          if (!x.c || !x.c[0] || x.c[0] == 1 && !x.e && x.c.length == 1 || !n2.c || !n2.c[0]) {
            y = new BigNumber2(Math.pow(+valueOf(x), nIsBig ? n2.s * (2 - isOdd(n2)) : +valueOf(n2)));
            return m ? y.mod(m) : y;
          }
          nIsNeg = n2.s < 0;
          if (m) {
            if (m.c ? !m.c[0] : !m.s) return new BigNumber2(NaN);
            isModExp = !nIsNeg && x.isInteger() && m.isInteger();
            if (isModExp) x = x.mod(m);
          } else if (n2.e > 9 && (x.e > 0 || x.e < -1 || (x.e == 0 ? x.c[0] > 1 || nIsBig && x.c[1] >= 24e7 : x.c[0] < 8e13 || nIsBig && x.c[0] <= 9999975e7))) {
            k = x.s < 0 && isOdd(n2) ? -0 : 0;
            if (x.e > -1) k = 1 / k;
            return new BigNumber2(nIsNeg ? 1 / k : k);
          } else if (POW_PRECISION) {
            k = mathceil(POW_PRECISION / LOG_BASE + 2);
          }
          if (nIsBig) {
            half = new BigNumber2(0.5);
            if (nIsNeg) n2.s = 1;
            nIsOdd = isOdd(n2);
          } else {
            i3 = Math.abs(+valueOf(n2));
            nIsOdd = i3 % 2;
          }
          y = new BigNumber2(ONE);
          for (; ; ) {
            if (nIsOdd) {
              y = y.times(x);
              if (!y.c) break;
              if (k) {
                if (y.c.length > k) y.c.length = k;
              } else if (isModExp) {
                y = y.mod(m);
              }
            }
            if (i3) {
              i3 = mathfloor(i3 / 2);
              if (i3 === 0) break;
              nIsOdd = i3 % 2;
            } else {
              n2 = n2.times(half);
              round(n2, n2.e + 1, 1);
              if (n2.e > 14) {
                nIsOdd = isOdd(n2);
              } else {
                i3 = +valueOf(n2);
                if (i3 === 0) break;
                nIsOdd = i3 % 2;
              }
            }
            x = x.times(x);
            if (k) {
              if (x.c && x.c.length > k) x.c.length = k;
            } else if (isModExp) {
              x = x.mod(m);
            }
          }
          if (isModExp) return y;
          if (nIsNeg) y = ONE.div(y);
          return m ? y.mod(m) : k ? round(y, POW_PRECISION, ROUNDING_MODE, more) : y;
        };
        P.integerValue = function(rm) {
          var n2 = new BigNumber2(this);
          if (rm == null) rm = ROUNDING_MODE;
          else intCheck(rm, 0, 8);
          return round(n2, n2.e + 1, rm);
        };
        P.isEqualTo = P.eq = function(y, b) {
          return compare(this, new BigNumber2(y, b)) === 0;
        };
        P.isFinite = function() {
          return !!this.c;
        };
        P.isGreaterThan = P.gt = function(y, b) {
          return compare(this, new BigNumber2(y, b)) > 0;
        };
        P.isGreaterThanOrEqualTo = P.gte = function(y, b) {
          return (b = compare(this, new BigNumber2(y, b))) === 1 || b === 0;
        };
        P.isInteger = function() {
          return !!this.c && bitFloor(this.e / LOG_BASE) > this.c.length - 2;
        };
        P.isLessThan = P.lt = function(y, b) {
          return compare(this, new BigNumber2(y, b)) < 0;
        };
        P.isLessThanOrEqualTo = P.lte = function(y, b) {
          return (b = compare(this, new BigNumber2(y, b))) === -1 || b === 0;
        };
        P.isNaN = function() {
          return !this.s;
        };
        P.isNegative = function() {
          return this.s < 0;
        };
        P.isPositive = function() {
          return this.s > 0;
        };
        P.isZero = function() {
          return !!this.c && this.c[0] == 0;
        };
        P.minus = function(y, b) {
          var i3, j, t2, xLTy, x = this, a3 = x.s;
          y = new BigNumber2(y, b);
          b = y.s;
          if (!a3 || !b) return new BigNumber2(NaN);
          if (a3 != b) {
            y.s = -b;
            return x.plus(y);
          }
          var xe = x.e / LOG_BASE, ye = y.e / LOG_BASE, xc = x.c, yc = y.c;
          if (!xe || !ye) {
            if (!xc || !yc) return xc ? (y.s = -b, y) : new BigNumber2(yc ? x : NaN);
            if (!xc[0] || !yc[0]) {
              return yc[0] ? (y.s = -b, y) : new BigNumber2(xc[0] ? x : (
                // IEEE 754 (2008) 6.3: n - n = -0 when rounding to -Infinity
                ROUNDING_MODE == 3 ? -0 : 0
              ));
            }
          }
          xe = bitFloor(xe);
          ye = bitFloor(ye);
          xc = xc.slice();
          if (a3 = xe - ye) {
            if (xLTy = a3 < 0) {
              a3 = -a3;
              t2 = xc;
            } else {
              ye = xe;
              t2 = yc;
            }
            t2.reverse();
            for (b = a3; b--; t2.push(0)) ;
            t2.reverse();
          } else {
            j = (xLTy = (a3 = xc.length) < (b = yc.length)) ? a3 : b;
            for (a3 = b = 0; b < j; b++) {
              if (xc[b] != yc[b]) {
                xLTy = xc[b] < yc[b];
                break;
              }
            }
          }
          if (xLTy) {
            t2 = xc;
            xc = yc;
            yc = t2;
            y.s = -y.s;
          }
          b = (j = yc.length) - (i3 = xc.length);
          if (b > 0) for (; b--; xc[i3++] = 0) ;
          b = BASE - 1;
          for (; j > a3; ) {
            if (xc[--j] < yc[j]) {
              for (i3 = j; i3 && !xc[--i3]; xc[i3] = b) ;
              --xc[i3];
              xc[j] += BASE;
            }
            xc[j] -= yc[j];
          }
          for (; xc[0] == 0; xc.splice(0, 1), --ye) ;
          if (!xc[0]) {
            y.s = ROUNDING_MODE == 3 ? -1 : 1;
            y.c = [y.e = 0];
            return y;
          }
          return normalise(y, xc, ye);
        };
        P.modulo = P.mod = function(y, b) {
          var q, s2, x = this;
          y = new BigNumber2(y, b);
          if (!x.c || !y.s || y.c && !y.c[0]) {
            return new BigNumber2(NaN);
          } else if (!y.c || x.c && !x.c[0]) {
            return new BigNumber2(x);
          }
          if (MODULO_MODE == 9) {
            s2 = y.s;
            y.s = 1;
            q = div(x, y, 0, 3);
            y.s = s2;
            q.s *= s2;
          } else {
            q = div(x, y, 0, MODULO_MODE);
          }
          y = x.minus(q.times(y));
          if (!y.c[0] && MODULO_MODE == 1) y.s = x.s;
          return y;
        };
        P.multipliedBy = P.times = function(y, b) {
          var c, e3, i3, j, k, m, xcL, xlo, xhi, ycL, ylo, yhi, zc, base, sqrtBase, x = this, xc = x.c, yc = (y = new BigNumber2(y, b)).c;
          if (!xc || !yc || !xc[0] || !yc[0]) {
            if (!x.s || !y.s || xc && !xc[0] && !yc || yc && !yc[0] && !xc) {
              y.c = y.e = y.s = null;
            } else {
              y.s *= x.s;
              if (!xc || !yc) {
                y.c = y.e = null;
              } else {
                y.c = [0];
                y.e = 0;
              }
            }
            return y;
          }
          e3 = bitFloor(x.e / LOG_BASE) + bitFloor(y.e / LOG_BASE);
          y.s *= x.s;
          xcL = xc.length;
          ycL = yc.length;
          if (xcL < ycL) {
            zc = xc;
            xc = yc;
            yc = zc;
            i3 = xcL;
            xcL = ycL;
            ycL = i3;
          }
          for (i3 = xcL + ycL, zc = []; i3--; zc.push(0)) ;
          base = BASE;
          sqrtBase = SQRT_BASE;
          for (i3 = ycL; --i3 >= 0; ) {
            c = 0;
            ylo = yc[i3] % sqrtBase;
            yhi = yc[i3] / sqrtBase | 0;
            for (k = xcL, j = i3 + k; j > i3; ) {
              xlo = xc[--k] % sqrtBase;
              xhi = xc[k] / sqrtBase | 0;
              m = yhi * xlo + xhi * ylo;
              xlo = ylo * xlo + m % sqrtBase * sqrtBase + zc[j] + c;
              c = (xlo / base | 0) + (m / sqrtBase | 0) + yhi * xhi;
              zc[j--] = xlo % base;
            }
            zc[j] = c;
          }
          if (c) {
            ++e3;
          } else {
            zc.splice(0, 1);
          }
          return normalise(y, zc, e3);
        };
        P.negated = function() {
          var x = new BigNumber2(this);
          x.s = -x.s || null;
          return x;
        };
        P.plus = function(y, b) {
          var t2, x = this, a3 = x.s;
          y = new BigNumber2(y, b);
          b = y.s;
          if (!a3 || !b) return new BigNumber2(NaN);
          if (a3 != b) {
            y.s = -b;
            return x.minus(y);
          }
          var xe = x.e / LOG_BASE, ye = y.e / LOG_BASE, xc = x.c, yc = y.c;
          if (!xe || !ye) {
            if (!xc || !yc) return new BigNumber2(a3 / 0);
            if (!xc[0] || !yc[0]) return yc[0] ? y : new BigNumber2(xc[0] ? x : a3 * 0);
          }
          xe = bitFloor(xe);
          ye = bitFloor(ye);
          xc = xc.slice();
          if (a3 = xe - ye) {
            if (a3 > 0) {
              ye = xe;
              t2 = yc;
            } else {
              a3 = -a3;
              t2 = xc;
            }
            t2.reverse();
            for (; a3--; t2.push(0)) ;
            t2.reverse();
          }
          a3 = xc.length;
          b = yc.length;
          if (a3 - b < 0) {
            t2 = yc;
            yc = xc;
            xc = t2;
            b = a3;
          }
          for (a3 = 0; b; ) {
            a3 = (xc[--b] = xc[b] + yc[b] + a3) / BASE | 0;
            xc[b] = BASE === xc[b] ? 0 : xc[b] % BASE;
          }
          if (a3) {
            xc = [a3].concat(xc);
            ++ye;
          }
          return normalise(y, xc, ye);
        };
        P.precision = P.sd = function(sd, rm) {
          var c, n2, v, x = this;
          if (sd != null && sd !== !!sd) {
            intCheck(sd, 1, MAX);
            if (rm == null) rm = ROUNDING_MODE;
            else intCheck(rm, 0, 8);
            return round(new BigNumber2(x), sd, rm);
          }
          if (!(c = x.c)) return null;
          v = c.length - 1;
          n2 = v * LOG_BASE + 1;
          if (v = c[v]) {
            for (; v % 10 == 0; v /= 10, n2--) ;
            for (v = c[0]; v >= 10; v /= 10, n2++) ;
          }
          if (sd && x.e + 1 > n2) n2 = x.e + 1;
          return n2;
        };
        P.shiftedBy = function(k) {
          intCheck(k, -MAX_SAFE_INTEGER, MAX_SAFE_INTEGER);
          return this.times("1e" + k);
        };
        P.squareRoot = P.sqrt = function() {
          var m, n2, r2, rep, t2, x = this, c = x.c, s2 = x.s, e3 = x.e, dp = DECIMAL_PLACES + 4, half = new BigNumber2("0.5");
          if (s2 !== 1 || !c || !c[0]) {
            return new BigNumber2(!s2 || s2 < 0 && (!c || c[0]) ? NaN : c ? x : 1 / 0);
          }
          s2 = Math.sqrt(+valueOf(x));
          if (s2 == 0 || s2 == 1 / 0) {
            n2 = coeffToString(c);
            if ((n2.length + e3) % 2 == 0) n2 += "0";
            s2 = Math.sqrt(+n2);
            e3 = bitFloor((e3 + 1) / 2) - (e3 < 0 || e3 % 2);
            if (s2 == 1 / 0) {
              n2 = "5e" + e3;
            } else {
              n2 = s2.toExponential();
              n2 = n2.slice(0, n2.indexOf("e") + 1) + e3;
            }
            r2 = new BigNumber2(n2);
          } else {
            r2 = new BigNumber2(s2 + "");
          }
          if (r2.c[0]) {
            e3 = r2.e;
            s2 = e3 + dp;
            if (s2 < 3) s2 = 0;
            for (; ; ) {
              t2 = r2;
              r2 = half.times(t2.plus(div(x, t2, dp, 1)));
              if (coeffToString(t2.c).slice(0, s2) === (n2 = coeffToString(r2.c)).slice(0, s2)) {
                if (r2.e < e3) --s2;
                n2 = n2.slice(s2 - 3, s2 + 1);
                if (n2 == "9999" || !rep && n2 == "4999") {
                  if (!rep) {
                    round(t2, t2.e + DECIMAL_PLACES + 2, 0);
                    if (t2.times(t2).eq(x)) {
                      r2 = t2;
                      break;
                    }
                  }
                  dp += 4;
                  s2 += 4;
                  rep = 1;
                } else {
                  if (!+n2 || !+n2.slice(1) && n2.charAt(0) == "5") {
                    round(r2, r2.e + DECIMAL_PLACES + 2, 1);
                    m = !r2.times(r2).eq(x);
                  }
                  break;
                }
              }
            }
          }
          return round(r2, r2.e + DECIMAL_PLACES + 1, ROUNDING_MODE, m);
        };
        P.toExponential = function(dp, rm) {
          if (dp != null) {
            intCheck(dp, 0, MAX);
            dp++;
          }
          return format(this, dp, rm, 1);
        };
        P.toFixed = function(dp, rm) {
          if (dp != null) {
            intCheck(dp, 0, MAX);
            dp = dp + this.e + 1;
          }
          return format(this, dp, rm);
        };
        P.toFormat = function(dp, rm, format2) {
          var str, x = this;
          if (format2 == null) {
            if (dp != null && rm && typeof rm == "object") {
              format2 = rm;
              rm = null;
            } else if (dp && typeof dp == "object") {
              format2 = dp;
              dp = rm = null;
            } else {
              format2 = FORMAT;
            }
          } else if (typeof format2 != "object") {
            throw Error(bignumberError + "Argument not an object: " + format2);
          }
          str = x.toFixed(dp, rm);
          if (x.c) {
            var i3, arr = str.split("."), g1 = +format2.groupSize, g2 = +format2.secondaryGroupSize, groupSeparator = format2.groupSeparator || "", intPart = arr[0], fractionPart = arr[1], isNeg = x.s < 0, intDigits = isNeg ? intPart.slice(1) : intPart, len = intDigits.length;
            if (g2) {
              i3 = g1;
              g1 = g2;
              g2 = i3;
              len -= i3;
            }
            if (g1 > 0 && len > 0) {
              i3 = len % g1 || g1;
              intPart = intDigits.substr(0, i3);
              for (; i3 < len; i3 += g1) intPart += groupSeparator + intDigits.substr(i3, g1);
              if (g2 > 0) intPart += groupSeparator + intDigits.slice(i3);
              if (isNeg) intPart = "-" + intPart;
            }
            str = fractionPart ? intPart + (format2.decimalSeparator || "") + ((g2 = +format2.fractionGroupSize) ? fractionPart.replace(
              new RegExp("\\d{" + g2 + "}\\B", "g"),
              "$&" + (format2.fractionGroupSeparator || "")
            ) : fractionPart) : intPart;
          }
          return (format2.prefix || "") + str + (format2.suffix || "");
        };
        P.toFraction = function(md) {
          var d, d0, d1, d2, e3, exp, n2, n0, n1, q, r2, s2, x = this, xc = x.c;
          if (md != null) {
            n2 = new BigNumber2(md);
            if (!n2.isInteger() && (n2.c || n2.s !== 1) || n2.lt(ONE)) {
              throw Error(bignumberError + "Argument " + (n2.isInteger() ? "out of range: " : "not an integer: ") + valueOf(n2));
            }
          }
          if (!xc) return new BigNumber2(x);
          d = new BigNumber2(ONE);
          n1 = d0 = new BigNumber2(ONE);
          d1 = n0 = new BigNumber2(ONE);
          s2 = coeffToString(xc);
          e3 = d.e = s2.length - x.e - 1;
          d.c[0] = POWS_TEN[(exp = e3 % LOG_BASE) < 0 ? LOG_BASE + exp : exp];
          md = !md || n2.comparedTo(d) > 0 ? e3 > 0 ? d : n1 : n2;
          exp = MAX_EXP;
          MAX_EXP = 1 / 0;
          n2 = new BigNumber2(s2);
          n0.c[0] = 0;
          for (; ; ) {
            q = div(n2, d, 0, 1);
            d2 = d0.plus(q.times(d1));
            if (d2.comparedTo(md) == 1) break;
            d0 = d1;
            d1 = d2;
            n1 = n0.plus(q.times(d2 = n1));
            n0 = d2;
            d = n2.minus(q.times(d2 = d));
            n2 = d2;
          }
          d2 = div(md.minus(d0), d1, 0, 1);
          n0 = n0.plus(d2.times(n1));
          d0 = d0.plus(d2.times(d1));
          n0.s = n1.s = x.s;
          e3 = e3 * 2;
          r2 = div(n1, d1, e3, ROUNDING_MODE).minus(x).abs().comparedTo(
            div(n0, d0, e3, ROUNDING_MODE).minus(x).abs()
          ) < 1 ? [n1, d1] : [n0, d0];
          MAX_EXP = exp;
          return r2;
        };
        P.toNumber = function() {
          return +valueOf(this);
        };
        P.toPrecision = function(sd, rm) {
          if (sd != null) intCheck(sd, 1, MAX);
          return format(this, sd, rm, 2);
        };
        P.toString = function(b) {
          var str, n2 = this, s2 = n2.s, e3 = n2.e;
          if (e3 === null) {
            if (s2) {
              str = "Infinity";
              if (s2 < 0) str = "-" + str;
            } else {
              str = "NaN";
            }
          } else {
            if (b == null) {
              str = e3 <= TO_EXP_NEG || e3 >= TO_EXP_POS ? toExponential(coeffToString(n2.c), e3) : toFixedPoint(coeffToString(n2.c), e3, "0");
            } else if (b === 10 && alphabetHasNormalDecimalDigits) {
              n2 = round(new BigNumber2(n2), DECIMAL_PLACES + e3 + 1, ROUNDING_MODE);
              str = toFixedPoint(coeffToString(n2.c), n2.e, "0");
            } else {
              intCheck(b, 2, ALPHABET.length, "Base");
              str = convertBase(toFixedPoint(coeffToString(n2.c), e3, "0"), 10, b, s2, true);
            }
            if (s2 < 0 && n2.c[0]) str = "-" + str;
          }
          return str;
        };
        P.valueOf = P.toJSON = function() {
          return valueOf(this);
        };
        P._isBigNumber = true;
        if (configObject != null) BigNumber2.set(configObject);
        return BigNumber2;
      }
      __name(clone, "clone");
      function bitFloor(n2) {
        var i3 = n2 | 0;
        return n2 > 0 || n2 === i3 ? i3 : i3 - 1;
      }
      __name(bitFloor, "bitFloor");
      function coeffToString(a3) {
        var s2, z, i3 = 1, j = a3.length, r2 = a3[0] + "";
        for (; i3 < j; ) {
          s2 = a3[i3++] + "";
          z = LOG_BASE - s2.length;
          for (; z--; s2 = "0" + s2) ;
          r2 += s2;
        }
        for (j = r2.length; r2.charCodeAt(--j) === 48; ) ;
        return r2.slice(0, j + 1 || 1);
      }
      __name(coeffToString, "coeffToString");
      function compare(x, y) {
        var a3, b, xc = x.c, yc = y.c, i3 = x.s, j = y.s, k = x.e, l2 = y.e;
        if (!i3 || !j) return null;
        a3 = xc && !xc[0];
        b = yc && !yc[0];
        if (a3 || b) return a3 ? b ? 0 : -j : i3;
        if (i3 != j) return i3;
        a3 = i3 < 0;
        b = k == l2;
        if (!xc || !yc) return b ? 0 : !xc ^ a3 ? 1 : -1;
        if (!b) return k > l2 ^ a3 ? 1 : -1;
        j = (k = xc.length) < (l2 = yc.length) ? k : l2;
        for (i3 = 0; i3 < j; i3++) if (xc[i3] != yc[i3]) return xc[i3] > yc[i3] ^ a3 ? 1 : -1;
        return k == l2 ? 0 : k > l2 ^ a3 ? 1 : -1;
      }
      __name(compare, "compare");
      function intCheck(n2, min, max, name) {
        if (n2 < min || n2 > max || n2 !== mathfloor(n2)) {
          throw Error(bignumberError + (name || "Argument") + (typeof n2 == "number" ? n2 < min || n2 > max ? " out of range: " : " not an integer: " : " not a primitive number: ") + String(n2));
        }
      }
      __name(intCheck, "intCheck");
      function isOdd(n2) {
        var k = n2.c.length - 1;
        return bitFloor(n2.e / LOG_BASE) == k && n2.c[k] % 2 != 0;
      }
      __name(isOdd, "isOdd");
      function toExponential(str, e3) {
        return (str.length > 1 ? str.charAt(0) + "." + str.slice(1) : str) + (e3 < 0 ? "e" : "e+") + e3;
      }
      __name(toExponential, "toExponential");
      function toFixedPoint(str, e3, z) {
        var len, zs;
        if (e3 < 0) {
          for (zs = z + "."; ++e3; zs += z) ;
          str = zs + str;
        } else {
          len = str.length;
          if (++e3 > len) {
            for (zs = z, e3 -= len; --e3; zs += z) ;
            str += zs;
          } else if (e3 < len) {
            str = str.slice(0, e3) + "." + str.slice(e3);
          }
        }
        return str;
      }
      __name(toFixedPoint, "toFixedPoint");
      BigNumber = clone();
      BigNumber["default"] = BigNumber.BigNumber = BigNumber;
      if (typeof define == "function" && define.amd) {
        define(function() {
          return BigNumber;
        });
      } else if (typeof module2 != "undefined" && module2.exports) {
        module2.exports = BigNumber;
      } else {
        if (!globalObject) {
          globalObject = typeof self != "undefined" && self ? self : window;
        }
        globalObject.BigNumber = BigNumber;
      }
    })(exports);
  }
});

// node_modules/json-bigint/lib/stringify.js
var require_stringify = __commonJS({
  "node_modules/json-bigint/lib/stringify.js"(exports, module2) {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    var BigNumber = require_bignumber();
    var JSON2 = module2.exports;
    (function() {
      "use strict";
      function f(n2) {
        return n2 < 10 ? "0" + n2 : n2;
      }
      __name(f, "f");
      var cx = /[\u0000\u00ad\u0600-\u0604\u070f\u17b4\u17b5\u200c-\u200f\u2028-\u202f\u2060-\u206f\ufeff\ufff0-\uffff]/g, escapable = /[\\\"\x00-\x1f\x7f-\x9f\u00ad\u0600-\u0604\u070f\u17b4\u17b5\u200c-\u200f\u2028-\u202f\u2060-\u206f\ufeff\ufff0-\uffff]/g, gap, indent, meta = {
        // table of character substitutions
        "\b": "\\b",
        "	": "\\t",
        "\n": "\\n",
        "\f": "\\f",
        "\r": "\\r",
        '"': '\\"',
        "\\": "\\\\"
      }, rep;
      function quote(string) {
        escapable.lastIndex = 0;
        return escapable.test(string) ? '"' + string.replace(escapable, function(a3) {
          var c = meta[a3];
          return typeof c === "string" ? c : "\\u" + ("0000" + a3.charCodeAt(0).toString(16)).slice(-4);
        }) + '"' : '"' + string + '"';
      }
      __name(quote, "quote");
      function str(key, holder) {
        var i3, k, v, length, mind = gap, partial, value = holder[key], isBigNumber2 = value != null && (value instanceof BigNumber || BigNumber.isBigNumber(value));
        if (value && typeof value === "object" && typeof value.toJSON === "function") {
          value = value.toJSON(key);
        }
        if (typeof rep === "function") {
          value = rep.call(holder, key, value);
        }
        switch (typeof value) {
          case "string":
            if (isBigNumber2) {
              return value;
            } else {
              return quote(value);
            }
          case "number":
            return isFinite(value) ? String(value) : "null";
          case "boolean":
          case "null":
          case "bigint":
            return String(value);
          // If the type is 'object', we might be dealing with an object or an array or
          // null.
          case "object":
            if (!value) {
              return "null";
            }
            gap += indent;
            partial = [];
            if (Object.prototype.toString.apply(value) === "[object Array]") {
              length = value.length;
              for (i3 = 0; i3 < length; i3 += 1) {
                partial[i3] = str(i3, value) || "null";
              }
              v = partial.length === 0 ? "[]" : gap ? "[\n" + gap + partial.join(",\n" + gap) + "\n" + mind + "]" : "[" + partial.join(",") + "]";
              gap = mind;
              return v;
            }
            if (rep && typeof rep === "object") {
              length = rep.length;
              for (i3 = 0; i3 < length; i3 += 1) {
                if (typeof rep[i3] === "string") {
                  k = rep[i3];
                  v = str(k, value);
                  if (v) {
                    partial.push(quote(k) + (gap ? ": " : ":") + v);
                  }
                }
              }
            } else {
              Object.keys(value).forEach(function(k2) {
                var v2 = str(k2, value);
                if (v2) {
                  partial.push(quote(k2) + (gap ? ": " : ":") + v2);
                }
              });
            }
            v = partial.length === 0 ? "{}" : gap ? "{\n" + gap + partial.join(",\n" + gap) + "\n" + mind + "}" : "{" + partial.join(",") + "}";
            gap = mind;
            return v;
        }
      }
      __name(str, "str");
      if (typeof JSON2.stringify !== "function") {
        JSON2.stringify = function(value, replacer, space) {
          var i3;
          gap = "";
          indent = "";
          if (typeof space === "number") {
            for (i3 = 0; i3 < space; i3 += 1) {
              indent += " ";
            }
          } else if (typeof space === "string") {
            indent = space;
          }
          rep = replacer;
          if (replacer && typeof replacer !== "function" && (typeof replacer !== "object" || typeof replacer.length !== "number")) {
            throw new Error("JSON.stringify");
          }
          return str("", { "": value });
        };
      }
    })();
  }
});

// node_modules/json-bigint/lib/parse.js
var require_parse = __commonJS({
  "node_modules/json-bigint/lib/parse.js"(exports, module2) {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    var BigNumber = null;
    var suspectProtoRx = /(?:_|\\u005[Ff])(?:_|\\u005[Ff])(?:p|\\u0070)(?:r|\\u0072)(?:o|\\u006[Ff])(?:t|\\u0074)(?:o|\\u006[Ff])(?:_|\\u005[Ff])(?:_|\\u005[Ff])/;
    var suspectConstructorRx = /(?:c|\\u0063)(?:o|\\u006[Ff])(?:n|\\u006[Ee])(?:s|\\u0073)(?:t|\\u0074)(?:r|\\u0072)(?:u|\\u0075)(?:c|\\u0063)(?:t|\\u0074)(?:o|\\u006[Ff])(?:r|\\u0072)/;
    var json_parse = /* @__PURE__ */ __name(function(options) {
      "use strict";
      var _options = {
        strict: false,
        // not being strict means do not generate syntax errors for "duplicate key"
        storeAsString: false,
        // toggles whether the values should be stored as BigNumber (default) or a string
        alwaysParseAsBig: false,
        // toggles whether all numbers should be Big
        useNativeBigInt: false,
        // toggles whether to use native BigInt instead of bignumber.js
        protoAction: "error",
        constructorAction: "error"
      };
      if (options !== void 0 && options !== null) {
        if (options.strict === true) {
          _options.strict = true;
        }
        if (options.storeAsString === true) {
          _options.storeAsString = true;
        }
        _options.alwaysParseAsBig = options.alwaysParseAsBig === true ? options.alwaysParseAsBig : false;
        _options.useNativeBigInt = options.useNativeBigInt === true ? options.useNativeBigInt : false;
        if (typeof options.constructorAction !== "undefined") {
          if (options.constructorAction === "error" || options.constructorAction === "ignore" || options.constructorAction === "preserve") {
            _options.constructorAction = options.constructorAction;
          } else {
            throw new Error(
              `Incorrect value for constructorAction option, must be "error", "ignore" or undefined but passed ${options.constructorAction}`
            );
          }
        }
        if (typeof options.protoAction !== "undefined") {
          if (options.protoAction === "error" || options.protoAction === "ignore" || options.protoAction === "preserve") {
            _options.protoAction = options.protoAction;
          } else {
            throw new Error(
              `Incorrect value for protoAction option, must be "error", "ignore" or undefined but passed ${options.protoAction}`
            );
          }
        }
      }
      var at, ch, escapee = {
        '"': '"',
        "\\": "\\",
        "/": "/",
        b: "\b",
        f: "\f",
        n: "\n",
        r: "\r",
        t: "	"
      }, text, error3 = /* @__PURE__ */ __name(function(m) {
        throw {
          name: "SyntaxError",
          message: m,
          at,
          text
        };
      }, "error"), next = /* @__PURE__ */ __name(function(c) {
        if (c && c !== ch) {
          error3("Expected '" + c + "' instead of '" + ch + "'");
        }
        ch = text.charAt(at);
        at += 1;
        return ch;
      }, "next"), number = /* @__PURE__ */ __name(function() {
        var number2, string2 = "";
        if (ch === "-") {
          string2 = "-";
          next("-");
        }
        while (ch >= "0" && ch <= "9") {
          string2 += ch;
          next();
        }
        if (ch === ".") {
          string2 += ".";
          while (next() && ch >= "0" && ch <= "9") {
            string2 += ch;
          }
        }
        if (ch === "e" || ch === "E") {
          string2 += ch;
          next();
          if (ch === "-" || ch === "+") {
            string2 += ch;
            next();
          }
          while (ch >= "0" && ch <= "9") {
            string2 += ch;
            next();
          }
        }
        number2 = +string2;
        if (!isFinite(number2)) {
          error3("Bad number");
        } else {
          if (BigNumber == null) BigNumber = require_bignumber();
          if (string2.length > 15)
            return _options.storeAsString ? string2 : _options.useNativeBigInt ? BigInt(string2) : new BigNumber(string2);
          else
            return !_options.alwaysParseAsBig ? number2 : _options.useNativeBigInt ? BigInt(number2) : new BigNumber(number2);
        }
      }, "number"), string = /* @__PURE__ */ __name(function() {
        var hex, i3, string2 = "", uffff;
        if (ch === '"') {
          var startAt = at;
          while (next()) {
            if (ch === '"') {
              if (at - 1 > startAt) string2 += text.substring(startAt, at - 1);
              next();
              return string2;
            }
            if (ch === "\\") {
              if (at - 1 > startAt) string2 += text.substring(startAt, at - 1);
              next();
              if (ch === "u") {
                uffff = 0;
                for (i3 = 0; i3 < 4; i3 += 1) {
                  hex = parseInt(next(), 16);
                  if (!isFinite(hex)) {
                    break;
                  }
                  uffff = uffff * 16 + hex;
                }
                string2 += String.fromCharCode(uffff);
              } else if (typeof escapee[ch] === "string") {
                string2 += escapee[ch];
              } else {
                break;
              }
              startAt = at;
            }
          }
        }
        error3("Bad string");
      }, "string"), white = /* @__PURE__ */ __name(function() {
        while (ch && ch <= " ") {
          next();
        }
      }, "white"), word = /* @__PURE__ */ __name(function() {
        switch (ch) {
          case "t":
            next("t");
            next("r");
            next("u");
            next("e");
            return true;
          case "f":
            next("f");
            next("a");
            next("l");
            next("s");
            next("e");
            return false;
          case "n":
            next("n");
            next("u");
            next("l");
            next("l");
            return null;
        }
        error3("Unexpected '" + ch + "'");
      }, "word"), value, array = /* @__PURE__ */ __name(function() {
        var array2 = [];
        if (ch === "[") {
          next("[");
          white();
          if (ch === "]") {
            next("]");
            return array2;
          }
          while (ch) {
            array2.push(value());
            white();
            if (ch === "]") {
              next("]");
              return array2;
            }
            next(",");
            white();
          }
        }
        error3("Bad array");
      }, "array"), object = /* @__PURE__ */ __name(function() {
        var key, object2 = /* @__PURE__ */ Object.create(null);
        if (ch === "{") {
          next("{");
          white();
          if (ch === "}") {
            next("}");
            return object2;
          }
          while (ch) {
            key = string();
            white();
            next(":");
            if (_options.strict === true && Object.hasOwnProperty.call(object2, key)) {
              error3('Duplicate key "' + key + '"');
            }
            if (suspectProtoRx.test(key) === true) {
              if (_options.protoAction === "error") {
                error3("Object contains forbidden prototype property");
              } else if (_options.protoAction === "ignore") {
                value();
              } else {
                object2[key] = value();
              }
            } else if (suspectConstructorRx.test(key) === true) {
              if (_options.constructorAction === "error") {
                error3("Object contains forbidden constructor property");
              } else if (_options.constructorAction === "ignore") {
                value();
              } else {
                object2[key] = value();
              }
            } else {
              object2[key] = value();
            }
            white();
            if (ch === "}") {
              next("}");
              return object2;
            }
            next(",");
            white();
          }
        }
        error3("Bad object");
      }, "object");
      value = /* @__PURE__ */ __name(function() {
        white();
        switch (ch) {
          case "{":
            return object();
          case "[":
            return array();
          case '"':
            return string();
          case "-":
            return number();
          default:
            return ch >= "0" && ch <= "9" ? number() : word();
        }
      }, "value");
      return function(source, reviver2) {
        var result;
        text = source + "";
        at = 0;
        ch = " ";
        result = value();
        white();
        if (ch) {
          error3("Syntax error");
        }
        return typeof reviver2 === "function" ? (/* @__PURE__ */ __name((function walk(holder, key) {
          var k, v, value2 = holder[key];
          if (value2 && typeof value2 === "object") {
            Object.keys(value2).forEach(function(k2) {
              v = walk(value2, k2);
              if (v !== void 0) {
                value2[k2] = v;
              } else {
                delete value2[k2];
              }
            });
          }
          return reviver2.call(holder, key, value2);
        }), "walk"))({ "": result }, "") : result;
      };
    }, "json_parse");
    module2.exports = json_parse;
  }
});

// node_modules/json-bigint/index.js
var require_json_bigint = __commonJS({
  "node_modules/json-bigint/index.js"(exports, module2) {
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
    init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
    init_performance2();
    var json_stringify = require_stringify().stringify;
    var json_parse = require_parse();
    module2.exports = function(options) {
      return {
        parse: json_parse(options),
        stringify: json_stringify
      };
    };
    module2.exports.parse = json_parse();
    module2.exports.stringify = json_stringify;
  }
});

// src/worker.ts
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/index.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/hono.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/hono-base.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/compose.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var compose = /* @__PURE__ */ __name((middleware, onError, onNotFound) => {
  return (context2, next) => {
    let index = -1;
    return dispatch(0);
    async function dispatch(i3) {
      if (i3 <= index) {
        throw new Error("next() called multiple times");
      }
      index = i3;
      let res;
      let isError = false;
      let handler;
      if (middleware[i3]) {
        handler = middleware[i3][0][0];
        context2.req.routeIndex = i3;
      } else {
        handler = i3 === middleware.length && next || void 0;
      }
      if (handler) {
        try {
          res = await handler(context2, () => dispatch(i3 + 1));
        } catch (err) {
          if (err instanceof Error && onError) {
            context2.error = err;
            res = await onError(err, context2);
            isError = true;
          } else {
            throw err;
          }
        }
      } else {
        if (context2.finalized === false && onNotFound) {
          res = await onNotFound(context2);
        }
      }
      if (res && (context2.finalized === false || isError)) {
        context2.res = res;
      }
      return context2;
    }
    __name(dispatch, "dispatch");
  };
}, "compose");

// node_modules/hono/dist/context.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/request.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/http-exception.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/request/constants.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var GET_MATCH_RESULT = /* @__PURE__ */ Symbol();

// node_modules/hono/dist/utils/body.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var parseBody = /* @__PURE__ */ __name(async (request, options = /* @__PURE__ */ Object.create(null)) => {
  const { all = false, dot = false } = options;
  const headers = request instanceof HonoRequest ? request.raw.headers : request.headers;
  const contentType = headers.get("Content-Type");
  if (contentType?.startsWith("multipart/form-data") || contentType?.startsWith("application/x-www-form-urlencoded")) {
    return parseFormData(request, { all, dot });
  }
  return {};
}, "parseBody");
async function parseFormData(request, options) {
  const formData = await request.formData();
  if (formData) {
    return convertFormDataToBodyData(formData, options);
  }
  return {};
}
__name(parseFormData, "parseFormData");
function convertFormDataToBodyData(formData, options) {
  const form = /* @__PURE__ */ Object.create(null);
  formData.forEach((value, key) => {
    const shouldParseAllValues = options.all || key.endsWith("[]");
    if (!shouldParseAllValues) {
      form[key] = value;
    } else {
      handleParsingAllValues(form, key, value);
    }
  });
  if (options.dot) {
    Object.entries(form).forEach(([key, value]) => {
      const shouldParseDotValues = key.includes(".");
      if (shouldParseDotValues) {
        handleParsingNestedValues(form, key, value);
        delete form[key];
      }
    });
  }
  return form;
}
__name(convertFormDataToBodyData, "convertFormDataToBodyData");
var handleParsingAllValues = /* @__PURE__ */ __name((form, key, value) => {
  if (form[key] !== void 0) {
    if (Array.isArray(form[key])) {
      ;
      form[key].push(value);
    } else {
      form[key] = [form[key], value];
    }
  } else {
    if (!key.endsWith("[]")) {
      form[key] = value;
    } else {
      form[key] = [value];
    }
  }
}, "handleParsingAllValues");
var handleParsingNestedValues = /* @__PURE__ */ __name((form, key, value) => {
  let nestedForm = form;
  const keys = key.split(".");
  keys.forEach((key2, index) => {
    if (index === keys.length - 1) {
      nestedForm[key2] = value;
    } else {
      if (!nestedForm[key2] || typeof nestedForm[key2] !== "object" || Array.isArray(nestedForm[key2]) || nestedForm[key2] instanceof File) {
        nestedForm[key2] = /* @__PURE__ */ Object.create(null);
      }
      nestedForm = nestedForm[key2];
    }
  });
}, "handleParsingNestedValues");

// node_modules/hono/dist/utils/url.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var splitPath = /* @__PURE__ */ __name((path) => {
  const paths = path.split("/");
  if (paths[0] === "") {
    paths.shift();
  }
  return paths;
}, "splitPath");
var splitRoutingPath = /* @__PURE__ */ __name((routePath) => {
  const { groups, path } = extractGroupsFromPath(routePath);
  const paths = splitPath(path);
  return replaceGroupMarks(paths, groups);
}, "splitRoutingPath");
var extractGroupsFromPath = /* @__PURE__ */ __name((path) => {
  const groups = [];
  path = path.replace(/\{[^}]+\}/g, (match2, index) => {
    const mark = `@${index}`;
    groups.push([mark, match2]);
    return mark;
  });
  return { groups, path };
}, "extractGroupsFromPath");
var replaceGroupMarks = /* @__PURE__ */ __name((paths, groups) => {
  for (let i3 = groups.length - 1; i3 >= 0; i3--) {
    const [mark] = groups[i3];
    for (let j = paths.length - 1; j >= 0; j--) {
      if (paths[j].includes(mark)) {
        paths[j] = paths[j].replace(mark, groups[i3][1]);
        break;
      }
    }
  }
  return paths;
}, "replaceGroupMarks");
var patternCache = {};
var getPattern = /* @__PURE__ */ __name((label, next) => {
  if (label === "*") {
    return "*";
  }
  const match2 = label.match(/^\:([^\{\}]+)(?:\{(.+)\})?$/);
  if (match2) {
    const cacheKey = `${label}#${next}`;
    if (!patternCache[cacheKey]) {
      if (match2[2]) {
        patternCache[cacheKey] = next && next[0] !== ":" && next[0] !== "*" ? [cacheKey, match2[1], new RegExp(`^${match2[2]}(?=/${next})`)] : [label, match2[1], new RegExp(`^${match2[2]}$`)];
      } else {
        patternCache[cacheKey] = [label, match2[1], true];
      }
    }
    return patternCache[cacheKey];
  }
  return null;
}, "getPattern");
var tryDecode = /* @__PURE__ */ __name((str, decoder) => {
  try {
    return decoder(str);
  } catch {
    return str.replace(/(?:%[0-9A-Fa-f]{2})+/g, (match2) => {
      try {
        return decoder(match2);
      } catch {
        return match2;
      }
    });
  }
}, "tryDecode");
var tryDecodeURI = /* @__PURE__ */ __name((str) => tryDecode(str, decodeURI), "tryDecodeURI");
var getPath = /* @__PURE__ */ __name((request) => {
  const url = request.url;
  const start = url.indexOf("/", url.indexOf(":") + 4);
  let i3 = start;
  for (; i3 < url.length; i3++) {
    const charCode = url.charCodeAt(i3);
    if (charCode === 37) {
      const queryIndex = url.indexOf("?", i3);
      const hashIndex = url.indexOf("#", i3);
      const end = queryIndex === -1 ? hashIndex === -1 ? void 0 : hashIndex : hashIndex === -1 ? queryIndex : Math.min(queryIndex, hashIndex);
      const path = url.slice(start, end);
      return tryDecodeURI(path.includes("%25") ? path.replace(/%25/g, "%2525") : path);
    } else if (charCode === 63 || charCode === 35) {
      break;
    }
  }
  return url.slice(start, i3);
}, "getPath");
var getPathNoStrict = /* @__PURE__ */ __name((request) => {
  const result = getPath(request);
  return result.length > 1 && result.at(-1) === "/" ? result.slice(0, -1) : result;
}, "getPathNoStrict");
var mergePath = /* @__PURE__ */ __name((base, sub, ...rest) => {
  if (rest.length) {
    sub = mergePath(sub, ...rest);
  }
  return `${base?.[0] === "/" ? "" : "/"}${base}${sub === "/" ? "" : `${base?.at(-1) === "/" ? "" : "/"}${sub?.[0] === "/" ? sub.slice(1) : sub}`}`;
}, "mergePath");
var checkOptionalParameter = /* @__PURE__ */ __name((path) => {
  if (path.charCodeAt(path.length - 1) !== 63 || !path.includes(":")) {
    return null;
  }
  const segments = path.split("/");
  const results = [];
  let basePath = "";
  segments.forEach((segment) => {
    if (segment !== "" && !/\:/.test(segment)) {
      basePath += "/" + segment;
    } else if (/\:/.test(segment)) {
      if (/\?/.test(segment)) {
        if (results.length === 0 && basePath === "") {
          results.push("/");
        } else {
          results.push(basePath);
        }
        const optionalSegment = segment.replace("?", "");
        basePath += "/" + optionalSegment;
        results.push(basePath);
      } else {
        basePath += "/" + segment;
      }
    }
  });
  return results.filter((v, i3, a3) => a3.indexOf(v) === i3);
}, "checkOptionalParameter");
var _decodeURI = /* @__PURE__ */ __name((value) => {
  if (!/[%+]/.test(value)) {
    return value;
  }
  if (value.indexOf("+") !== -1) {
    value = value.replace(/\+/g, " ");
  }
  return value.indexOf("%") !== -1 ? tryDecode(value, decodeURIComponent_) : value;
}, "_decodeURI");
var _getQueryParam = /* @__PURE__ */ __name((url, key, multiple) => {
  let encoded;
  if (!multiple && key && !/[%+]/.test(key)) {
    let keyIndex2 = url.indexOf("?", 8);
    if (keyIndex2 === -1) {
      return void 0;
    }
    if (!url.startsWith(key, keyIndex2 + 1)) {
      keyIndex2 = url.indexOf(`&${key}`, keyIndex2 + 1);
    }
    while (keyIndex2 !== -1) {
      const trailingKeyCode = url.charCodeAt(keyIndex2 + key.length + 1);
      if (trailingKeyCode === 61) {
        const valueIndex = keyIndex2 + key.length + 2;
        const endIndex = url.indexOf("&", valueIndex);
        return _decodeURI(url.slice(valueIndex, endIndex === -1 ? void 0 : endIndex));
      } else if (trailingKeyCode == 38 || isNaN(trailingKeyCode)) {
        return "";
      }
      keyIndex2 = url.indexOf(`&${key}`, keyIndex2 + 1);
    }
    encoded = /[%+]/.test(url);
    if (!encoded) {
      return void 0;
    }
  }
  const results = {};
  encoded ??= /[%+]/.test(url);
  let keyIndex = url.indexOf("?", 8);
  while (keyIndex !== -1) {
    const nextKeyIndex = url.indexOf("&", keyIndex + 1);
    let valueIndex = url.indexOf("=", keyIndex);
    if (valueIndex > nextKeyIndex && nextKeyIndex !== -1) {
      valueIndex = -1;
    }
    let name = url.slice(
      keyIndex + 1,
      valueIndex === -1 ? nextKeyIndex === -1 ? void 0 : nextKeyIndex : valueIndex
    );
    if (encoded) {
      name = _decodeURI(name);
    }
    keyIndex = nextKeyIndex;
    if (name === "") {
      continue;
    }
    let value;
    if (valueIndex === -1) {
      value = "";
    } else {
      value = url.slice(valueIndex + 1, nextKeyIndex === -1 ? void 0 : nextKeyIndex);
      if (encoded) {
        value = _decodeURI(value);
      }
    }
    if (multiple) {
      if (!(results[name] && Array.isArray(results[name]))) {
        results[name] = [];
      }
      ;
      results[name].push(value);
    } else {
      results[name] ??= value;
    }
  }
  return key ? results[key] : results;
}, "_getQueryParam");
var getQueryParam = _getQueryParam;
var getQueryParams = /* @__PURE__ */ __name((url, key) => {
  return _getQueryParam(url, key, true);
}, "getQueryParams");
var decodeURIComponent_ = decodeURIComponent;

// node_modules/hono/dist/request.js
var tryDecodeURIComponent = /* @__PURE__ */ __name((str) => tryDecode(str, decodeURIComponent_), "tryDecodeURIComponent");
var HonoRequest = class {
  static {
    __name(this, "HonoRequest");
  }
  /**
   * `.raw` can get the raw Request object.
   *
   * @see {@link https://hono.dev/docs/api/request#raw}
   *
   * @example
   * ```ts
   * // For Cloudflare Workers
   * app.post('/', async (c) => {
   *   const metadata = c.req.raw.cf?.hostMetadata?
   *   ...
   * })
   * ```
   */
  raw;
  #validatedData;
  // Short name of validatedData
  #matchResult;
  routeIndex = 0;
  /**
   * `.path` can get the pathname of the request.
   *
   * @see {@link https://hono.dev/docs/api/request#path}
   *
   * @example
   * ```ts
   * app.get('/about/me', (c) => {
   *   const pathname = c.req.path // `/about/me`
   * })
   * ```
   */
  path;
  bodyCache = {};
  constructor(request, path = "/", matchResult = [[]]) {
    this.raw = request;
    this.path = path;
    this.#matchResult = matchResult;
    this.#validatedData = {};
  }
  param(key) {
    return key ? this.#getDecodedParam(key) : this.#getAllDecodedParams();
  }
  #getDecodedParam(key) {
    const paramKey = this.#matchResult[0][this.routeIndex][1][key];
    const param = this.#getParamValue(paramKey);
    return param && /\%/.test(param) ? tryDecodeURIComponent(param) : param;
  }
  #getAllDecodedParams() {
    const decoded = {};
    const keys = Object.keys(this.#matchResult[0][this.routeIndex][1]);
    for (const key of keys) {
      const value = this.#getParamValue(this.#matchResult[0][this.routeIndex][1][key]);
      if (value !== void 0) {
        decoded[key] = /\%/.test(value) ? tryDecodeURIComponent(value) : value;
      }
    }
    return decoded;
  }
  #getParamValue(paramKey) {
    return this.#matchResult[1] ? this.#matchResult[1][paramKey] : paramKey;
  }
  query(key) {
    return getQueryParam(this.url, key);
  }
  queries(key) {
    return getQueryParams(this.url, key);
  }
  header(name) {
    if (name) {
      return this.raw.headers.get(name) ?? void 0;
    }
    const headerData = {};
    this.raw.headers.forEach((value, key) => {
      headerData[key] = value;
    });
    return headerData;
  }
  async parseBody(options) {
    return this.bodyCache.parsedBody ??= await parseBody(this, options);
  }
  #cachedBody = /* @__PURE__ */ __name((key) => {
    const { bodyCache, raw: raw2 } = this;
    const cachedBody = bodyCache[key];
    if (cachedBody) {
      return cachedBody;
    }
    const anyCachedKey = Object.keys(bodyCache)[0];
    if (anyCachedKey) {
      return bodyCache[anyCachedKey].then((body) => {
        if (anyCachedKey === "json") {
          body = JSON.stringify(body);
        }
        return new Response(body)[key]();
      });
    }
    return bodyCache[key] = raw2[key]();
  }, "#cachedBody");
  /**
   * `.json()` can parse Request body of type `application/json`
   *
   * @see {@link https://hono.dev/docs/api/request#json}
   *
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.json()
   * })
   * ```
   */
  json() {
    return this.#cachedBody("text").then((text) => JSON.parse(text));
  }
  /**
   * `.text()` can parse Request body of type `text/plain`
   *
   * @see {@link https://hono.dev/docs/api/request#text}
   *
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.text()
   * })
   * ```
   */
  text() {
    return this.#cachedBody("text");
  }
  /**
   * `.arrayBuffer()` parse Request body as an `ArrayBuffer`
   *
   * @see {@link https://hono.dev/docs/api/request#arraybuffer}
   *
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.arrayBuffer()
   * })
   * ```
   */
  arrayBuffer() {
    return this.#cachedBody("arrayBuffer");
  }
  /**
   * Parses the request body as a `Blob`.
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.blob();
   * });
   * ```
   * @see https://hono.dev/docs/api/request#blob
   */
  blob() {
    return this.#cachedBody("blob");
  }
  /**
   * Parses the request body as `FormData`.
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.formData();
   * });
   * ```
   * @see https://hono.dev/docs/api/request#formdata
   */
  formData() {
    return this.#cachedBody("formData");
  }
  /**
   * Adds validated data to the request.
   *
   * @param target - The target of the validation.
   * @param data - The validated data to add.
   */
  addValidatedData(target, data) {
    this.#validatedData[target] = data;
  }
  valid(target) {
    return this.#validatedData[target];
  }
  /**
   * `.url()` can get the request url strings.
   *
   * @see {@link https://hono.dev/docs/api/request#url}
   *
   * @example
   * ```ts
   * app.get('/about/me', (c) => {
   *   const url = c.req.url // `http://localhost:8787/about/me`
   *   ...
   * })
   * ```
   */
  get url() {
    return this.raw.url;
  }
  /**
   * `.method()` can get the method name of the request.
   *
   * @see {@link https://hono.dev/docs/api/request#method}
   *
   * @example
   * ```ts
   * app.get('/about/me', (c) => {
   *   const method = c.req.method // `GET`
   * })
   * ```
   */
  get method() {
    return this.raw.method;
  }
  get [GET_MATCH_RESULT]() {
    return this.#matchResult;
  }
  /**
   * `.matchedRoutes()` can return a matched route in the handler
   *
   * @deprecated
   *
   * Use matchedRoutes helper defined in "hono/route" instead.
   *
   * @see {@link https://hono.dev/docs/api/request#matchedroutes}
   *
   * @example
   * ```ts
   * app.use('*', async function logger(c, next) {
   *   await next()
   *   c.req.matchedRoutes.forEach(({ handler, method, path }, i) => {
   *     const name = handler.name || (handler.length < 2 ? '[handler]' : '[middleware]')
   *     console.log(
   *       method,
   *       ' ',
   *       path,
   *       ' '.repeat(Math.max(10 - path.length, 0)),
   *       name,
   *       i === c.req.routeIndex ? '<- respond from here' : ''
   *     )
   *   })
   * })
   * ```
   */
  get matchedRoutes() {
    return this.#matchResult[0].map(([[, route]]) => route);
  }
  /**
   * `routePath()` can retrieve the path registered within the handler
   *
   * @deprecated
   *
   * Use routePath helper defined in "hono/route" instead.
   *
   * @see {@link https://hono.dev/docs/api/request#routepath}
   *
   * @example
   * ```ts
   * app.get('/posts/:id', (c) => {
   *   return c.json({ path: c.req.routePath })
   * })
   * ```
   */
  get routePath() {
    return this.#matchResult[0].map(([[, route]]) => route)[this.routeIndex].path;
  }
};

// node_modules/hono/dist/utils/html.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var HtmlEscapedCallbackPhase = {
  Stringify: 1,
  BeforeStream: 2,
  Stream: 3
};
var raw = /* @__PURE__ */ __name((value, callbacks) => {
  const escapedString = new String(value);
  escapedString.isEscaped = true;
  escapedString.callbacks = callbacks;
  return escapedString;
}, "raw");
var resolveCallback = /* @__PURE__ */ __name(async (str, phase, preserveCallbacks, context2, buffer) => {
  if (typeof str === "object" && !(str instanceof String)) {
    if (!(str instanceof Promise)) {
      str = str.toString();
    }
    if (str instanceof Promise) {
      str = await str;
    }
  }
  const callbacks = str.callbacks;
  if (!callbacks?.length) {
    return Promise.resolve(str);
  }
  if (buffer) {
    buffer[0] += str;
  } else {
    buffer = [str];
  }
  const resStr = Promise.all(callbacks.map((c) => c({ phase, buffer, context: context2 }))).then(
    (res) => Promise.all(
      res.filter(Boolean).map((str2) => resolveCallback(str2, phase, false, context2, buffer))
    ).then(() => buffer[0])
  );
  if (preserveCallbacks) {
    return raw(await resStr, callbacks);
  } else {
    return resStr;
  }
}, "resolveCallback");

// node_modules/hono/dist/context.js
var TEXT_PLAIN = "text/plain; charset=UTF-8";
var setDefaultContentType = /* @__PURE__ */ __name((contentType, headers) => {
  return {
    "Content-Type": contentType,
    ...headers
  };
}, "setDefaultContentType");
var createResponseInstance = /* @__PURE__ */ __name((body, init) => new Response(body, init), "createResponseInstance");
var Context = class {
  static {
    __name(this, "Context");
  }
  #rawRequest;
  #req;
  /**
   * `.env` can get bindings (environment variables, secrets, KV namespaces, D1 database, R2 bucket etc.) in Cloudflare Workers.
   *
   * @see {@link https://hono.dev/docs/api/context#env}
   *
   * @example
   * ```ts
   * // Environment object for Cloudflare Workers
   * app.get('*', async c => {
   *   const counter = c.env.COUNTER
   * })
   * ```
   */
  env = {};
  #var;
  finalized = false;
  /**
   * `.error` can get the error object from the middleware if the Handler throws an error.
   *
   * @see {@link https://hono.dev/docs/api/context#error}
   *
   * @example
   * ```ts
   * app.use('*', async (c, next) => {
   *   await next()
   *   if (c.error) {
   *     // do something...
   *   }
   * })
   * ```
   */
  error;
  #status;
  #executionCtx;
  #res;
  #layout;
  #renderer;
  #notFoundHandler;
  #preparedHeaders;
  #matchResult;
  #path;
  /**
   * Creates an instance of the Context class.
   *
   * @param req - The Request object.
   * @param options - Optional configuration options for the context.
   */
  constructor(req, options) {
    this.#rawRequest = req;
    if (options) {
      this.#executionCtx = options.executionCtx;
      this.env = options.env;
      this.#notFoundHandler = options.notFoundHandler;
      this.#path = options.path;
      this.#matchResult = options.matchResult;
    }
  }
  /**
   * `.req` is the instance of {@link HonoRequest}.
   */
  get req() {
    this.#req ??= new HonoRequest(this.#rawRequest, this.#path, this.#matchResult);
    return this.#req;
  }
  /**
   * @see {@link https://hono.dev/docs/api/context#event}
   * The FetchEvent associated with the current request.
   *
   * @throws Will throw an error if the context does not have a FetchEvent.
   */
  get event() {
    if (this.#executionCtx && "respondWith" in this.#executionCtx) {
      return this.#executionCtx;
    } else {
      throw Error("This context has no FetchEvent");
    }
  }
  /**
   * @see {@link https://hono.dev/docs/api/context#executionctx}
   * The ExecutionContext associated with the current request.
   *
   * @throws Will throw an error if the context does not have an ExecutionContext.
   */
  get executionCtx() {
    if (this.#executionCtx) {
      return this.#executionCtx;
    } else {
      throw Error("This context has no ExecutionContext");
    }
  }
  /**
   * @see {@link https://hono.dev/docs/api/context#res}
   * The Response object for the current request.
   */
  get res() {
    return this.#res ||= createResponseInstance(null, {
      headers: this.#preparedHeaders ??= new Headers()
    });
  }
  /**
   * Sets the Response object for the current request.
   *
   * @param _res - The Response object to set.
   */
  set res(_res) {
    if (this.#res && _res) {
      _res = createResponseInstance(_res.body, _res);
      for (const [k, v] of this.#res.headers.entries()) {
        if (k === "content-type") {
          continue;
        }
        if (k === "set-cookie") {
          const cookies = this.#res.headers.getSetCookie();
          _res.headers.delete("set-cookie");
          for (const cookie of cookies) {
            _res.headers.append("set-cookie", cookie);
          }
        } else {
          _res.headers.set(k, v);
        }
      }
    }
    this.#res = _res;
    this.finalized = true;
  }
  /**
   * `.render()` can create a response within a layout.
   *
   * @see {@link https://hono.dev/docs/api/context#render-setrenderer}
   *
   * @example
   * ```ts
   * app.get('/', (c) => {
   *   return c.render('Hello!')
   * })
   * ```
   */
  render = /* @__PURE__ */ __name((...args) => {
    this.#renderer ??= (content) => this.html(content);
    return this.#renderer(...args);
  }, "render");
  /**
   * Sets the layout for the response.
   *
   * @param layout - The layout to set.
   * @returns The layout function.
   */
  setLayout = /* @__PURE__ */ __name((layout) => this.#layout = layout, "setLayout");
  /**
   * Gets the current layout for the response.
   *
   * @returns The current layout function.
   */
  getLayout = /* @__PURE__ */ __name(() => this.#layout, "getLayout");
  /**
   * `.setRenderer()` can set the layout in the custom middleware.
   *
   * @see {@link https://hono.dev/docs/api/context#render-setrenderer}
   *
   * @example
   * ```tsx
   * app.use('*', async (c, next) => {
   *   c.setRenderer((content) => {
   *     return c.html(
   *       <html>
   *         <body>
   *           <p>{content}</p>
   *         </body>
   *       </html>
   *     )
   *   })
   *   await next()
   * })
   * ```
   */
  setRenderer = /* @__PURE__ */ __name((renderer) => {
    this.#renderer = renderer;
  }, "setRenderer");
  /**
   * `.header()` can set headers.
   *
   * @see {@link https://hono.dev/docs/api/context#header}
   *
   * @example
   * ```ts
   * app.get('/welcome', (c) => {
   *   // Set headers
   *   c.header('X-Message', 'Hello!')
   *   c.header('Content-Type', 'text/plain')
   *
   *   return c.body('Thank you for coming')
   * })
   * ```
   */
  header = /* @__PURE__ */ __name((name, value, options) => {
    if (this.finalized) {
      this.#res = createResponseInstance(this.#res.body, this.#res);
    }
    const headers = this.#res ? this.#res.headers : this.#preparedHeaders ??= new Headers();
    if (value === void 0) {
      headers.delete(name);
    } else if (options?.append) {
      headers.append(name, value);
    } else {
      headers.set(name, value);
    }
  }, "header");
  status = /* @__PURE__ */ __name((status) => {
    this.#status = status;
  }, "status");
  /**
   * `.set()` can set the value specified by the key.
   *
   * @see {@link https://hono.dev/docs/api/context#set-get}
   *
   * @example
   * ```ts
   * app.use('*', async (c, next) => {
   *   c.set('message', 'Hono is hot!!')
   *   await next()
   * })
   * ```
   */
  set = /* @__PURE__ */ __name((key, value) => {
    this.#var ??= /* @__PURE__ */ new Map();
    this.#var.set(key, value);
  }, "set");
  /**
   * `.get()` can use the value specified by the key.
   *
   * @see {@link https://hono.dev/docs/api/context#set-get}
   *
   * @example
   * ```ts
   * app.get('/', (c) => {
   *   const message = c.get('message')
   *   return c.text(`The message is "${message}"`)
   * })
   * ```
   */
  get = /* @__PURE__ */ __name((key) => {
    return this.#var ? this.#var.get(key) : void 0;
  }, "get");
  /**
   * `.var` can access the value of a variable.
   *
   * @see {@link https://hono.dev/docs/api/context#var}
   *
   * @example
   * ```ts
   * const result = c.var.client.oneMethod()
   * ```
   */
  // c.var.propName is a read-only
  get var() {
    if (!this.#var) {
      return {};
    }
    return Object.fromEntries(this.#var);
  }
  #newResponse(data, arg, headers) {
    const responseHeaders = this.#res ? new Headers(this.#res.headers) : this.#preparedHeaders ?? new Headers();
    if (typeof arg === "object" && "headers" in arg) {
      const argHeaders = arg.headers instanceof Headers ? arg.headers : new Headers(arg.headers);
      for (const [key, value] of argHeaders) {
        if (key.toLowerCase() === "set-cookie") {
          responseHeaders.append(key, value);
        } else {
          responseHeaders.set(key, value);
        }
      }
    }
    if (headers) {
      for (const [k, v] of Object.entries(headers)) {
        if (typeof v === "string") {
          responseHeaders.set(k, v);
        } else {
          responseHeaders.delete(k);
          for (const v2 of v) {
            responseHeaders.append(k, v2);
          }
        }
      }
    }
    const status = typeof arg === "number" ? arg : arg?.status ?? this.#status;
    return createResponseInstance(data, { status, headers: responseHeaders });
  }
  newResponse = /* @__PURE__ */ __name((...args) => this.#newResponse(...args), "newResponse");
  /**
   * `.body()` can return the HTTP response.
   * You can set headers with `.header()` and set HTTP status code with `.status`.
   * This can also be set in `.text()`, `.json()` and so on.
   *
   * @see {@link https://hono.dev/docs/api/context#body}
   *
   * @example
   * ```ts
   * app.get('/welcome', (c) => {
   *   // Set headers
   *   c.header('X-Message', 'Hello!')
   *   c.header('Content-Type', 'text/plain')
   *   // Set HTTP status code
   *   c.status(201)
   *
   *   // Return the response body
   *   return c.body('Thank you for coming')
   * })
   * ```
   */
  body = /* @__PURE__ */ __name((data, arg, headers) => this.#newResponse(data, arg, headers), "body");
  /**
   * `.text()` can render text as `Content-Type:text/plain`.
   *
   * @see {@link https://hono.dev/docs/api/context#text}
   *
   * @example
   * ```ts
   * app.get('/say', (c) => {
   *   return c.text('Hello!')
   * })
   * ```
   */
  text = /* @__PURE__ */ __name((text, arg, headers) => {
    return !this.#preparedHeaders && !this.#status && !arg && !headers && !this.finalized ? new Response(text) : this.#newResponse(
      text,
      arg,
      setDefaultContentType(TEXT_PLAIN, headers)
    );
  }, "text");
  /**
   * `.json()` can render JSON as `Content-Type:application/json`.
   *
   * @see {@link https://hono.dev/docs/api/context#json}
   *
   * @example
   * ```ts
   * app.get('/api', (c) => {
   *   return c.json({ message: 'Hello!' })
   * })
   * ```
   */
  json = /* @__PURE__ */ __name((object, arg, headers) => {
    return this.#newResponse(
      JSON.stringify(object),
      arg,
      setDefaultContentType("application/json", headers)
    );
  }, "json");
  html = /* @__PURE__ */ __name((html, arg, headers) => {
    const res = /* @__PURE__ */ __name((html2) => this.#newResponse(html2, arg, setDefaultContentType("text/html; charset=UTF-8", headers)), "res");
    return typeof html === "object" ? resolveCallback(html, HtmlEscapedCallbackPhase.Stringify, false, {}).then(res) : res(html);
  }, "html");
  /**
   * `.redirect()` can Redirect, default status code is 302.
   *
   * @see {@link https://hono.dev/docs/api/context#redirect}
   *
   * @example
   * ```ts
   * app.get('/redirect', (c) => {
   *   return c.redirect('/')
   * })
   * app.get('/redirect-permanently', (c) => {
   *   return c.redirect('/', 301)
   * })
   * ```
   */
  redirect = /* @__PURE__ */ __name((location, status) => {
    const locationString = String(location);
    this.header(
      "Location",
      // Multibyes should be encoded
      // eslint-disable-next-line no-control-regex
      !/[^\x00-\xFF]/.test(locationString) ? locationString : encodeURI(locationString)
    );
    return this.newResponse(null, status ?? 302);
  }, "redirect");
  /**
   * `.notFound()` can return the Not Found Response.
   *
   * @see {@link https://hono.dev/docs/api/context#notfound}
   *
   * @example
   * ```ts
   * app.get('/notfound', (c) => {
   *   return c.notFound()
   * })
   * ```
   */
  notFound = /* @__PURE__ */ __name(() => {
    this.#notFoundHandler ??= () => createResponseInstance();
    return this.#notFoundHandler(this);
  }, "notFound");
};

// node_modules/hono/dist/router.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var METHOD_NAME_ALL = "ALL";
var METHOD_NAME_ALL_LOWERCASE = "all";
var METHODS = ["get", "post", "put", "delete", "options", "patch"];
var MESSAGE_MATCHER_IS_ALREADY_BUILT = "Can not add a route since the matcher is already built.";
var UnsupportedPathError = class extends Error {
  static {
    __name(this, "UnsupportedPathError");
  }
};

// node_modules/hono/dist/utils/constants.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var COMPOSED_HANDLER = "__COMPOSED_HANDLER";

// node_modules/hono/dist/hono-base.js
var notFoundHandler = /* @__PURE__ */ __name((c) => {
  return c.text("404 Not Found", 404);
}, "notFoundHandler");
var errorHandler = /* @__PURE__ */ __name((err, c) => {
  if ("getResponse" in err) {
    const res = err.getResponse();
    return c.newResponse(res.body, res);
  }
  console.error(err);
  return c.text("Internal Server Error", 500);
}, "errorHandler");
var Hono = class _Hono {
  static {
    __name(this, "_Hono");
  }
  get;
  post;
  put;
  delete;
  options;
  patch;
  all;
  on;
  use;
  /*
    This class is like an abstract class and does not have a router.
    To use it, inherit the class and implement router in the constructor.
  */
  router;
  getPath;
  // Cannot use `#` because it requires visibility at JavaScript runtime.
  _basePath = "/";
  #path = "/";
  routes = [];
  constructor(options = {}) {
    const allMethods = [...METHODS, METHOD_NAME_ALL_LOWERCASE];
    allMethods.forEach((method) => {
      this[method] = (args1, ...args) => {
        if (typeof args1 === "string") {
          this.#path = args1;
        } else {
          this.#addRoute(method, this.#path, args1);
        }
        args.forEach((handler) => {
          this.#addRoute(method, this.#path, handler);
        });
        return this;
      };
    });
    this.on = (method, path, ...handlers) => {
      for (const p of [path].flat()) {
        this.#path = p;
        for (const m of [method].flat()) {
          handlers.map((handler) => {
            this.#addRoute(m.toUpperCase(), this.#path, handler);
          });
        }
      }
      return this;
    };
    this.use = (arg1, ...handlers) => {
      if (typeof arg1 === "string") {
        this.#path = arg1;
      } else {
        this.#path = "*";
        handlers.unshift(arg1);
      }
      handlers.forEach((handler) => {
        this.#addRoute(METHOD_NAME_ALL, this.#path, handler);
      });
      return this;
    };
    const { strict, ...optionsWithoutStrict } = options;
    Object.assign(this, optionsWithoutStrict);
    this.getPath = strict ?? true ? options.getPath ?? getPath : getPathNoStrict;
  }
  #clone() {
    const clone = new _Hono({
      router: this.router,
      getPath: this.getPath
    });
    clone.errorHandler = this.errorHandler;
    clone.#notFoundHandler = this.#notFoundHandler;
    clone.routes = this.routes;
    return clone;
  }
  #notFoundHandler = notFoundHandler;
  // Cannot use `#` because it requires visibility at JavaScript runtime.
  errorHandler = errorHandler;
  /**
   * `.route()` allows grouping other Hono instance in routes.
   *
   * @see {@link https://hono.dev/docs/api/routing#grouping}
   *
   * @param {string} path - base Path
   * @param {Hono} app - other Hono instance
   * @returns {Hono} routed Hono instance
   *
   * @example
   * ```ts
   * const app = new Hono()
   * const app2 = new Hono()
   *
   * app2.get("/user", (c) => c.text("user"))
   * app.route("/api", app2) // GET /api/user
   * ```
   */
  route(path, app2) {
    const subApp = this.basePath(path);
    app2.routes.map((r2) => {
      let handler;
      if (app2.errorHandler === errorHandler) {
        handler = r2.handler;
      } else {
        handler = /* @__PURE__ */ __name(async (c, next) => (await compose([], app2.errorHandler)(c, () => r2.handler(c, next))).res, "handler");
        handler[COMPOSED_HANDLER] = r2.handler;
      }
      subApp.#addRoute(r2.method, r2.path, handler);
    });
    return this;
  }
  /**
   * `.basePath()` allows base paths to be specified.
   *
   * @see {@link https://hono.dev/docs/api/routing#base-path}
   *
   * @param {string} path - base Path
   * @returns {Hono} changed Hono instance
   *
   * @example
   * ```ts
   * const api = new Hono().basePath('/api')
   * ```
   */
  basePath(path) {
    const subApp = this.#clone();
    subApp._basePath = mergePath(this._basePath, path);
    return subApp;
  }
  /**
   * `.onError()` handles an error and returns a customized Response.
   *
   * @see {@link https://hono.dev/docs/api/hono#error-handling}
   *
   * @param {ErrorHandler} handler - request Handler for error
   * @returns {Hono} changed Hono instance
   *
   * @example
   * ```ts
   * app.onError((err, c) => {
   *   console.error(`${err}`)
   *   return c.text('Custom Error Message', 500)
   * })
   * ```
   */
  onError = /* @__PURE__ */ __name((handler) => {
    this.errorHandler = handler;
    return this;
  }, "onError");
  /**
   * `.notFound()` allows you to customize a Not Found Response.
   *
   * @see {@link https://hono.dev/docs/api/hono#not-found}
   *
   * @param {NotFoundHandler} handler - request handler for not-found
   * @returns {Hono} changed Hono instance
   *
   * @example
   * ```ts
   * app.notFound((c) => {
   *   return c.text('Custom 404 Message', 404)
   * })
   * ```
   */
  notFound = /* @__PURE__ */ __name((handler) => {
    this.#notFoundHandler = handler;
    return this;
  }, "notFound");
  /**
   * `.mount()` allows you to mount applications built with other frameworks into your Hono application.
   *
   * @see {@link https://hono.dev/docs/api/hono#mount}
   *
   * @param {string} path - base Path
   * @param {Function} applicationHandler - other Request Handler
   * @param {MountOptions} [options] - options of `.mount()`
   * @returns {Hono} mounted Hono instance
   *
   * @example
   * ```ts
   * import { Router as IttyRouter } from 'itty-router'
   * import { Hono } from 'hono'
   * // Create itty-router application
   * const ittyRouter = IttyRouter()
   * // GET /itty-router/hello
   * ittyRouter.get('/hello', () => new Response('Hello from itty-router'))
   *
   * const app = new Hono()
   * app.mount('/itty-router', ittyRouter.handle)
   * ```
   *
   * @example
   * ```ts
   * const app = new Hono()
   * // Send the request to another application without modification.
   * app.mount('/app', anotherApp, {
   *   replaceRequest: (req) => req,
   * })
   * ```
   */
  mount(path, applicationHandler, options) {
    let replaceRequest;
    let optionHandler;
    if (options) {
      if (typeof options === "function") {
        optionHandler = options;
      } else {
        optionHandler = options.optionHandler;
        if (options.replaceRequest === false) {
          replaceRequest = /* @__PURE__ */ __name((request) => request, "replaceRequest");
        } else {
          replaceRequest = options.replaceRequest;
        }
      }
    }
    const getOptions = optionHandler ? (c) => {
      const options2 = optionHandler(c);
      return Array.isArray(options2) ? options2 : [options2];
    } : (c) => {
      let executionContext = void 0;
      try {
        executionContext = c.executionCtx;
      } catch {
      }
      return [c.env, executionContext];
    };
    replaceRequest ||= (() => {
      const mergedPath = mergePath(this._basePath, path);
      const pathPrefixLength = mergedPath === "/" ? 0 : mergedPath.length;
      return (request) => {
        const url = new URL(request.url);
        url.pathname = url.pathname.slice(pathPrefixLength) || "/";
        return new Request(url, request);
      };
    })();
    const handler = /* @__PURE__ */ __name(async (c, next) => {
      const res = await applicationHandler(replaceRequest(c.req.raw), ...getOptions(c));
      if (res) {
        return res;
      }
      await next();
    }, "handler");
    this.#addRoute(METHOD_NAME_ALL, mergePath(path, "*"), handler);
    return this;
  }
  #addRoute(method, path, handler) {
    method = method.toUpperCase();
    path = mergePath(this._basePath, path);
    const r2 = { basePath: this._basePath, path, method, handler };
    this.router.add(method, path, [handler, r2]);
    this.routes.push(r2);
  }
  #handleError(err, c) {
    if (err instanceof Error) {
      return this.errorHandler(err, c);
    }
    throw err;
  }
  #dispatch(request, executionCtx, env2, method) {
    if (method === "HEAD") {
      return (async () => new Response(null, await this.#dispatch(request, executionCtx, env2, "GET")))();
    }
    const path = this.getPath(request, { env: env2 });
    const matchResult = this.router.match(method, path);
    const c = new Context(request, {
      path,
      matchResult,
      env: env2,
      executionCtx,
      notFoundHandler: this.#notFoundHandler
    });
    if (matchResult[0].length === 1) {
      let res;
      try {
        res = matchResult[0][0][0][0](c, async () => {
          c.res = await this.#notFoundHandler(c);
        });
      } catch (err) {
        return this.#handleError(err, c);
      }
      return res instanceof Promise ? res.then(
        (resolved) => resolved || (c.finalized ? c.res : this.#notFoundHandler(c))
      ).catch((err) => this.#handleError(err, c)) : res ?? this.#notFoundHandler(c);
    }
    const composed = compose(matchResult[0], this.errorHandler, this.#notFoundHandler);
    return (async () => {
      try {
        const context2 = await composed(c);
        if (!context2.finalized) {
          throw new Error(
            "Context is not finalized. Did you forget to return a Response object or `await next()`?"
          );
        }
        return context2.res;
      } catch (err) {
        return this.#handleError(err, c);
      }
    })();
  }
  /**
   * `.fetch()` will be entry point of your app.
   *
   * @see {@link https://hono.dev/docs/api/hono#fetch}
   *
   * @param {Request} request - request Object of request
   * @param {Env} Env - env Object
   * @param {ExecutionContext} - context of execution
   * @returns {Response | Promise<Response>} response of request
   *
   */
  fetch = /* @__PURE__ */ __name((request, ...rest) => {
    return this.#dispatch(request, rest[1], rest[0], request.method);
  }, "fetch");
  /**
   * `.request()` is a useful method for testing.
   * You can pass a URL or pathname to send a GET request.
   * app will return a Response object.
   * ```ts
   * test('GET /hello is ok', async () => {
   *   const res = await app.request('/hello')
   *   expect(res.status).toBe(200)
   * })
   * ```
   * @see https://hono.dev/docs/api/hono#request
   */
  request = /* @__PURE__ */ __name((input, requestInit, Env2, executionCtx) => {
    if (input instanceof Request) {
      return this.fetch(requestInit ? new Request(input, requestInit) : input, Env2, executionCtx);
    }
    input = input.toString();
    return this.fetch(
      new Request(
        /^https?:\/\//.test(input) ? input : `http://localhost${mergePath("/", input)}`,
        requestInit
      ),
      Env2,
      executionCtx
    );
  }, "request");
  /**
   * `.fire()` automatically adds a global fetch event listener.
   * This can be useful for environments that adhere to the Service Worker API, such as non-ES module Cloudflare Workers.
   * @deprecated
   * Use `fire` from `hono/service-worker` instead.
   * ```ts
   * import { Hono } from 'hono'
   * import { fire } from 'hono/service-worker'
   *
   * const app = new Hono()
   * // ...
   * fire(app)
   * ```
   * @see https://hono.dev/docs/api/hono#fire
   * @see https://developer.mozilla.org/en-US/docs/Web/API/Service_Worker_API
   * @see https://developers.cloudflare.com/workers/reference/migrate-to-module-workers/
   */
  fire = /* @__PURE__ */ __name(() => {
    addEventListener("fetch", (event) => {
      event.respondWith(this.#dispatch(event.request, event, void 0, event.request.method));
    });
  }, "fire");
};

// node_modules/hono/dist/router/reg-exp-router/index.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/router/reg-exp-router/router.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/router/reg-exp-router/matcher.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var emptyParam = [];
function match(method, path) {
  const matchers = this.buildAllMatchers();
  const match2 = /* @__PURE__ */ __name(((method2, path2) => {
    const matcher = matchers[method2] || matchers[METHOD_NAME_ALL];
    const staticMatch = matcher[2][path2];
    if (staticMatch) {
      return staticMatch;
    }
    const match3 = path2.match(matcher[0]);
    if (!match3) {
      return [[], emptyParam];
    }
    const index = match3.indexOf("", 1);
    return [matcher[1][index], match3];
  }), "match2");
  this.match = match2;
  return match2(method, path);
}
__name(match, "match");

// node_modules/hono/dist/router/reg-exp-router/node.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var LABEL_REG_EXP_STR = "[^/]+";
var ONLY_WILDCARD_REG_EXP_STR = ".*";
var TAIL_WILDCARD_REG_EXP_STR = "(?:|/.*)";
var PATH_ERROR = /* @__PURE__ */ Symbol();
var regExpMetaChars = new Set(".\\+*[^]$()");
function compareKey(a3, b) {
  if (a3.length === 1) {
    return b.length === 1 ? a3 < b ? -1 : 1 : -1;
  }
  if (b.length === 1) {
    return 1;
  }
  if (a3 === ONLY_WILDCARD_REG_EXP_STR || a3 === TAIL_WILDCARD_REG_EXP_STR) {
    return 1;
  } else if (b === ONLY_WILDCARD_REG_EXP_STR || b === TAIL_WILDCARD_REG_EXP_STR) {
    return -1;
  }
  if (a3 === LABEL_REG_EXP_STR) {
    return 1;
  } else if (b === LABEL_REG_EXP_STR) {
    return -1;
  }
  return a3.length === b.length ? a3 < b ? -1 : 1 : b.length - a3.length;
}
__name(compareKey, "compareKey");
var Node = class _Node {
  static {
    __name(this, "_Node");
  }
  #index;
  #varIndex;
  #children = /* @__PURE__ */ Object.create(null);
  insert(tokens, index, paramMap, context2, pathErrorCheckOnly) {
    if (tokens.length === 0) {
      if (this.#index !== void 0) {
        throw PATH_ERROR;
      }
      if (pathErrorCheckOnly) {
        return;
      }
      this.#index = index;
      return;
    }
    const [token, ...restTokens] = tokens;
    const pattern = token === "*" ? restTokens.length === 0 ? ["", "", ONLY_WILDCARD_REG_EXP_STR] : ["", "", LABEL_REG_EXP_STR] : token === "/*" ? ["", "", TAIL_WILDCARD_REG_EXP_STR] : token.match(/^\:([^\{\}]+)(?:\{(.+)\})?$/);
    let node;
    if (pattern) {
      const name = pattern[1];
      let regexpStr = pattern[2] || LABEL_REG_EXP_STR;
      if (name && pattern[2]) {
        if (regexpStr === ".*") {
          throw PATH_ERROR;
        }
        regexpStr = regexpStr.replace(/^\((?!\?:)(?=[^)]+\)$)/, "(?:");
        if (/\((?!\?:)/.test(regexpStr)) {
          throw PATH_ERROR;
        }
      }
      node = this.#children[regexpStr];
      if (!node) {
        if (Object.keys(this.#children).some(
          (k) => k !== ONLY_WILDCARD_REG_EXP_STR && k !== TAIL_WILDCARD_REG_EXP_STR
        )) {
          throw PATH_ERROR;
        }
        if (pathErrorCheckOnly) {
          return;
        }
        node = this.#children[regexpStr] = new _Node();
        if (name !== "") {
          node.#varIndex = context2.varIndex++;
        }
      }
      if (!pathErrorCheckOnly && name !== "") {
        paramMap.push([name, node.#varIndex]);
      }
    } else {
      node = this.#children[token];
      if (!node) {
        if (Object.keys(this.#children).some(
          (k) => k.length > 1 && k !== ONLY_WILDCARD_REG_EXP_STR && k !== TAIL_WILDCARD_REG_EXP_STR
        )) {
          throw PATH_ERROR;
        }
        if (pathErrorCheckOnly) {
          return;
        }
        node = this.#children[token] = new _Node();
      }
    }
    node.insert(restTokens, index, paramMap, context2, pathErrorCheckOnly);
  }
  buildRegExpStr() {
    const childKeys = Object.keys(this.#children).sort(compareKey);
    const strList = childKeys.map((k) => {
      const c = this.#children[k];
      return (typeof c.#varIndex === "number" ? `(${k})@${c.#varIndex}` : regExpMetaChars.has(k) ? `\\${k}` : k) + c.buildRegExpStr();
    });
    if (typeof this.#index === "number") {
      strList.unshift(`#${this.#index}`);
    }
    if (strList.length === 0) {
      return "";
    }
    if (strList.length === 1) {
      return strList[0];
    }
    return "(?:" + strList.join("|") + ")";
  }
};

// node_modules/hono/dist/router/reg-exp-router/trie.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var Trie = class {
  static {
    __name(this, "Trie");
  }
  #context = { varIndex: 0 };
  #root = new Node();
  insert(path, index, pathErrorCheckOnly) {
    const paramAssoc = [];
    const groups = [];
    for (let i3 = 0; ; ) {
      let replaced = false;
      path = path.replace(/\{[^}]+\}/g, (m) => {
        const mark = `@\\${i3}`;
        groups[i3] = [mark, m];
        i3++;
        replaced = true;
        return mark;
      });
      if (!replaced) {
        break;
      }
    }
    const tokens = path.match(/(?::[^\/]+)|(?:\/\*$)|./g) || [];
    for (let i3 = groups.length - 1; i3 >= 0; i3--) {
      const [mark] = groups[i3];
      for (let j = tokens.length - 1; j >= 0; j--) {
        if (tokens[j].indexOf(mark) !== -1) {
          tokens[j] = tokens[j].replace(mark, groups[i3][1]);
          break;
        }
      }
    }
    this.#root.insert(tokens, index, paramAssoc, this.#context, pathErrorCheckOnly);
    return paramAssoc;
  }
  buildRegExp() {
    let regexp = this.#root.buildRegExpStr();
    if (regexp === "") {
      return [/^$/, [], []];
    }
    let captureIndex = 0;
    const indexReplacementMap = [];
    const paramReplacementMap = [];
    regexp = regexp.replace(/#(\d+)|@(\d+)|\.\*\$/g, (_, handlerIndex, paramIndex) => {
      if (handlerIndex !== void 0) {
        indexReplacementMap[++captureIndex] = Number(handlerIndex);
        return "$()";
      }
      if (paramIndex !== void 0) {
        paramReplacementMap[Number(paramIndex)] = ++captureIndex;
        return "";
      }
      return "";
    });
    return [new RegExp(`^${regexp}`), indexReplacementMap, paramReplacementMap];
  }
};

// node_modules/hono/dist/router/reg-exp-router/router.js
var nullMatcher = [/^$/, [], /* @__PURE__ */ Object.create(null)];
var wildcardRegExpCache = /* @__PURE__ */ Object.create(null);
function buildWildcardRegExp(path) {
  return wildcardRegExpCache[path] ??= new RegExp(
    path === "*" ? "" : `^${path.replace(
      /\/\*$|([.\\+*[^\]$()])/g,
      (_, metaChar) => metaChar ? `\\${metaChar}` : "(?:|/.*)"
    )}$`
  );
}
__name(buildWildcardRegExp, "buildWildcardRegExp");
function clearWildcardRegExpCache() {
  wildcardRegExpCache = /* @__PURE__ */ Object.create(null);
}
__name(clearWildcardRegExpCache, "clearWildcardRegExpCache");
function buildMatcherFromPreprocessedRoutes(routes) {
  const trie = new Trie();
  const handlerData = [];
  if (routes.length === 0) {
    return nullMatcher;
  }
  const routesWithStaticPathFlag = routes.map(
    (route) => [!/\*|\/:/.test(route[0]), ...route]
  ).sort(
    ([isStaticA, pathA], [isStaticB, pathB]) => isStaticA ? 1 : isStaticB ? -1 : pathA.length - pathB.length
  );
  const staticMap = /* @__PURE__ */ Object.create(null);
  for (let i3 = 0, j = -1, len = routesWithStaticPathFlag.length; i3 < len; i3++) {
    const [pathErrorCheckOnly, path, handlers] = routesWithStaticPathFlag[i3];
    if (pathErrorCheckOnly) {
      staticMap[path] = [handlers.map(([h3]) => [h3, /* @__PURE__ */ Object.create(null)]), emptyParam];
    } else {
      j++;
    }
    let paramAssoc;
    try {
      paramAssoc = trie.insert(path, j, pathErrorCheckOnly);
    } catch (e3) {
      throw e3 === PATH_ERROR ? new UnsupportedPathError(path) : e3;
    }
    if (pathErrorCheckOnly) {
      continue;
    }
    handlerData[j] = handlers.map(([h3, paramCount]) => {
      const paramIndexMap = /* @__PURE__ */ Object.create(null);
      paramCount -= 1;
      for (; paramCount >= 0; paramCount--) {
        const [key, value] = paramAssoc[paramCount];
        paramIndexMap[key] = value;
      }
      return [h3, paramIndexMap];
    });
  }
  const [regexp, indexReplacementMap, paramReplacementMap] = trie.buildRegExp();
  for (let i3 = 0, len = handlerData.length; i3 < len; i3++) {
    for (let j = 0, len2 = handlerData[i3].length; j < len2; j++) {
      const map = handlerData[i3][j]?.[1];
      if (!map) {
        continue;
      }
      const keys = Object.keys(map);
      for (let k = 0, len3 = keys.length; k < len3; k++) {
        map[keys[k]] = paramReplacementMap[map[keys[k]]];
      }
    }
  }
  const handlerMap = [];
  for (const i3 in indexReplacementMap) {
    handlerMap[i3] = handlerData[indexReplacementMap[i3]];
  }
  return [regexp, handlerMap, staticMap];
}
__name(buildMatcherFromPreprocessedRoutes, "buildMatcherFromPreprocessedRoutes");
function findMiddleware(middleware, path) {
  if (!middleware) {
    return void 0;
  }
  for (const k of Object.keys(middleware).sort((a3, b) => b.length - a3.length)) {
    if (buildWildcardRegExp(k).test(path)) {
      return [...middleware[k]];
    }
  }
  return void 0;
}
__name(findMiddleware, "findMiddleware");
var RegExpRouter = class {
  static {
    __name(this, "RegExpRouter");
  }
  name = "RegExpRouter";
  #middleware;
  #routes;
  constructor() {
    this.#middleware = { [METHOD_NAME_ALL]: /* @__PURE__ */ Object.create(null) };
    this.#routes = { [METHOD_NAME_ALL]: /* @__PURE__ */ Object.create(null) };
  }
  add(method, path, handler) {
    const middleware = this.#middleware;
    const routes = this.#routes;
    if (!middleware || !routes) {
      throw new Error(MESSAGE_MATCHER_IS_ALREADY_BUILT);
    }
    if (!middleware[method]) {
      ;
      [middleware, routes].forEach((handlerMap) => {
        handlerMap[method] = /* @__PURE__ */ Object.create(null);
        Object.keys(handlerMap[METHOD_NAME_ALL]).forEach((p) => {
          handlerMap[method][p] = [...handlerMap[METHOD_NAME_ALL][p]];
        });
      });
    }
    if (path === "/*") {
      path = "*";
    }
    const paramCount = (path.match(/\/:/g) || []).length;
    if (/\*$/.test(path)) {
      const re = buildWildcardRegExp(path);
      if (method === METHOD_NAME_ALL) {
        Object.keys(middleware).forEach((m) => {
          middleware[m][path] ||= findMiddleware(middleware[m], path) || findMiddleware(middleware[METHOD_NAME_ALL], path) || [];
        });
      } else {
        middleware[method][path] ||= findMiddleware(middleware[method], path) || findMiddleware(middleware[METHOD_NAME_ALL], path) || [];
      }
      Object.keys(middleware).forEach((m) => {
        if (method === METHOD_NAME_ALL || method === m) {
          Object.keys(middleware[m]).forEach((p) => {
            re.test(p) && middleware[m][p].push([handler, paramCount]);
          });
        }
      });
      Object.keys(routes).forEach((m) => {
        if (method === METHOD_NAME_ALL || method === m) {
          Object.keys(routes[m]).forEach(
            (p) => re.test(p) && routes[m][p].push([handler, paramCount])
          );
        }
      });
      return;
    }
    const paths = checkOptionalParameter(path) || [path];
    for (let i3 = 0, len = paths.length; i3 < len; i3++) {
      const path2 = paths[i3];
      Object.keys(routes).forEach((m) => {
        if (method === METHOD_NAME_ALL || method === m) {
          routes[m][path2] ||= [
            ...findMiddleware(middleware[m], path2) || findMiddleware(middleware[METHOD_NAME_ALL], path2) || []
          ];
          routes[m][path2].push([handler, paramCount - len + i3 + 1]);
        }
      });
    }
  }
  match = match;
  buildAllMatchers() {
    const matchers = /* @__PURE__ */ Object.create(null);
    Object.keys(this.#routes).concat(Object.keys(this.#middleware)).forEach((method) => {
      matchers[method] ||= this.#buildMatcher(method);
    });
    this.#middleware = this.#routes = void 0;
    clearWildcardRegExpCache();
    return matchers;
  }
  #buildMatcher(method) {
    const routes = [];
    let hasOwnRoute = method === METHOD_NAME_ALL;
    [this.#middleware, this.#routes].forEach((r2) => {
      const ownRoute = r2[method] ? Object.keys(r2[method]).map((path) => [path, r2[method][path]]) : [];
      if (ownRoute.length !== 0) {
        hasOwnRoute ||= true;
        routes.push(...ownRoute);
      } else if (method !== METHOD_NAME_ALL) {
        routes.push(
          ...Object.keys(r2[METHOD_NAME_ALL]).map((path) => [path, r2[METHOD_NAME_ALL][path]])
        );
      }
    });
    if (!hasOwnRoute) {
      return null;
    } else {
      return buildMatcherFromPreprocessedRoutes(routes);
    }
  }
};

// node_modules/hono/dist/router/reg-exp-router/prepared-router.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/router/smart-router/index.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/router/smart-router/router.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var SmartRouter = class {
  static {
    __name(this, "SmartRouter");
  }
  name = "SmartRouter";
  #routers = [];
  #routes = [];
  constructor(init) {
    this.#routers = init.routers;
  }
  add(method, path, handler) {
    if (!this.#routes) {
      throw new Error(MESSAGE_MATCHER_IS_ALREADY_BUILT);
    }
    this.#routes.push([method, path, handler]);
  }
  match(method, path) {
    if (!this.#routes) {
      throw new Error("Fatal error");
    }
    const routers = this.#routers;
    const routes = this.#routes;
    const len = routers.length;
    let i3 = 0;
    let res;
    for (; i3 < len; i3++) {
      const router = routers[i3];
      try {
        for (let i22 = 0, len2 = routes.length; i22 < len2; i22++) {
          router.add(...routes[i22]);
        }
        res = router.match(method, path);
      } catch (e3) {
        if (e3 instanceof UnsupportedPathError) {
          continue;
        }
        throw e3;
      }
      this.match = router.match.bind(router);
      this.#routers = [router];
      this.#routes = void 0;
      break;
    }
    if (i3 === len) {
      throw new Error("Fatal error");
    }
    this.name = `SmartRouter + ${this.activeRouter.name}`;
    return res;
  }
  get activeRouter() {
    if (this.#routes || this.#routers.length !== 1) {
      throw new Error("No active router has been determined yet.");
    }
    return this.#routers[0];
  }
};

// node_modules/hono/dist/router/trie-router/index.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/router/trie-router/router.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/router/trie-router/node.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var emptyParams = /* @__PURE__ */ Object.create(null);
var hasChildren = /* @__PURE__ */ __name((children) => {
  for (const _ in children) {
    return true;
  }
  return false;
}, "hasChildren");
var Node2 = class _Node2 {
  static {
    __name(this, "_Node");
  }
  #methods;
  #children;
  #patterns;
  #order = 0;
  #params = emptyParams;
  constructor(method, handler, children) {
    this.#children = children || /* @__PURE__ */ Object.create(null);
    this.#methods = [];
    if (method && handler) {
      const m = /* @__PURE__ */ Object.create(null);
      m[method] = { handler, possibleKeys: [], score: 0 };
      this.#methods = [m];
    }
    this.#patterns = [];
  }
  insert(method, path, handler) {
    this.#order = ++this.#order;
    let curNode = this;
    const parts = splitRoutingPath(path);
    const possibleKeys = [];
    for (let i3 = 0, len = parts.length; i3 < len; i3++) {
      const p = parts[i3];
      const nextP = parts[i3 + 1];
      const pattern = getPattern(p, nextP);
      const key = Array.isArray(pattern) ? pattern[0] : p;
      if (key in curNode.#children) {
        curNode = curNode.#children[key];
        if (pattern) {
          possibleKeys.push(pattern[1]);
        }
        continue;
      }
      curNode.#children[key] = new _Node2();
      if (pattern) {
        curNode.#patterns.push(pattern);
        possibleKeys.push(pattern[1]);
      }
      curNode = curNode.#children[key];
    }
    curNode.#methods.push({
      [method]: {
        handler,
        possibleKeys: possibleKeys.filter((v, i3, a3) => a3.indexOf(v) === i3),
        score: this.#order
      }
    });
    return curNode;
  }
  #pushHandlerSets(handlerSets, node, method, nodeParams, params) {
    for (let i3 = 0, len = node.#methods.length; i3 < len; i3++) {
      const m = node.#methods[i3];
      const handlerSet = m[method] || m[METHOD_NAME_ALL];
      const processedSet = {};
      if (handlerSet !== void 0) {
        handlerSet.params = /* @__PURE__ */ Object.create(null);
        handlerSets.push(handlerSet);
        if (nodeParams !== emptyParams || params && params !== emptyParams) {
          for (let i22 = 0, len2 = handlerSet.possibleKeys.length; i22 < len2; i22++) {
            const key = handlerSet.possibleKeys[i22];
            const processed = processedSet[handlerSet.score];
            handlerSet.params[key] = params?.[key] && !processed ? params[key] : nodeParams[key] ?? params?.[key];
            processedSet[handlerSet.score] = true;
          }
        }
      }
    }
  }
  search(method, path) {
    const handlerSets = [];
    this.#params = emptyParams;
    const curNode = this;
    let curNodes = [curNode];
    const parts = splitPath(path);
    const curNodesQueue = [];
    const len = parts.length;
    let partOffsets = null;
    for (let i3 = 0; i3 < len; i3++) {
      const part = parts[i3];
      const isLast = i3 === len - 1;
      const tempNodes = [];
      for (let j = 0, len2 = curNodes.length; j < len2; j++) {
        const node = curNodes[j];
        const nextNode = node.#children[part];
        if (nextNode) {
          nextNode.#params = node.#params;
          if (isLast) {
            if (nextNode.#children["*"]) {
              this.#pushHandlerSets(handlerSets, nextNode.#children["*"], method, node.#params);
            }
            this.#pushHandlerSets(handlerSets, nextNode, method, node.#params);
          } else {
            tempNodes.push(nextNode);
          }
        }
        for (let k = 0, len3 = node.#patterns.length; k < len3; k++) {
          const pattern = node.#patterns[k];
          const params = node.#params === emptyParams ? {} : { ...node.#params };
          if (pattern === "*") {
            const astNode = node.#children["*"];
            if (astNode) {
              this.#pushHandlerSets(handlerSets, astNode, method, node.#params);
              astNode.#params = params;
              tempNodes.push(astNode);
            }
            continue;
          }
          const [key, name, matcher] = pattern;
          if (!part && !(matcher instanceof RegExp)) {
            continue;
          }
          const child = node.#children[key];
          if (matcher instanceof RegExp) {
            if (partOffsets === null) {
              partOffsets = new Array(len);
              let offset = path[0] === "/" ? 1 : 0;
              for (let p = 0; p < len; p++) {
                partOffsets[p] = offset;
                offset += parts[p].length + 1;
              }
            }
            const restPathString = path.substring(partOffsets[i3]);
            const m = matcher.exec(restPathString);
            if (m) {
              params[name] = m[0];
              this.#pushHandlerSets(handlerSets, child, method, node.#params, params);
              if (hasChildren(child.#children)) {
                child.#params = params;
                const componentCount = m[0].match(/\//)?.length ?? 0;
                const targetCurNodes = curNodesQueue[componentCount] ||= [];
                targetCurNodes.push(child);
              }
              continue;
            }
          }
          if (matcher === true || matcher.test(part)) {
            params[name] = part;
            if (isLast) {
              this.#pushHandlerSets(handlerSets, child, method, params, node.#params);
              if (child.#children["*"]) {
                this.#pushHandlerSets(
                  handlerSets,
                  child.#children["*"],
                  method,
                  params,
                  node.#params
                );
              }
            } else {
              child.#params = params;
              tempNodes.push(child);
            }
          }
        }
      }
      const shifted = curNodesQueue.shift();
      curNodes = shifted ? tempNodes.concat(shifted) : tempNodes;
    }
    if (handlerSets.length > 1) {
      handlerSets.sort((a3, b) => {
        return a3.score - b.score;
      });
    }
    return [handlerSets.map(({ handler, params }) => [handler, params])];
  }
};

// node_modules/hono/dist/router/trie-router/router.js
var TrieRouter = class {
  static {
    __name(this, "TrieRouter");
  }
  name = "TrieRouter";
  #node;
  constructor() {
    this.#node = new Node2();
  }
  add(method, path, handler) {
    const results = checkOptionalParameter(path);
    if (results) {
      for (let i3 = 0, len = results.length; i3 < len; i3++) {
        this.#node.insert(method, results[i3], handler);
      }
      return;
    }
    this.#node.insert(method, path, handler);
  }
  match(method, path) {
    return this.#node.search(method, path);
  }
};

// node_modules/hono/dist/hono.js
var Hono2 = class extends Hono {
  static {
    __name(this, "Hono");
  }
  /**
   * Creates an instance of the Hono class.
   *
   * @param options - Optional configuration options for the Hono instance.
   */
  constructor(options = {}) {
    super(options);
    this.router = options.router ?? new SmartRouter({
      routers: [new RegExpRouter(), new TrieRouter()]
    });
  }
};

// node_modules/hono/dist/middleware/cors/index.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var cors = /* @__PURE__ */ __name((options) => {
  const defaults = {
    origin: "*",
    allowMethods: ["GET", "HEAD", "PUT", "POST", "DELETE", "PATCH"],
    allowHeaders: [],
    exposeHeaders: []
  };
  const opts = {
    ...defaults,
    ...options
  };
  const findAllowOrigin = ((optsOrigin) => {
    if (typeof optsOrigin === "string") {
      if (optsOrigin === "*") {
        return () => optsOrigin;
      } else {
        return (origin) => optsOrigin === origin ? origin : null;
      }
    } else if (typeof optsOrigin === "function") {
      return optsOrigin;
    } else {
      return (origin) => optsOrigin.includes(origin) ? origin : null;
    }
  })(opts.origin);
  const findAllowMethods = ((optsAllowMethods) => {
    if (typeof optsAllowMethods === "function") {
      return optsAllowMethods;
    } else if (Array.isArray(optsAllowMethods)) {
      return () => optsAllowMethods;
    } else {
      return () => [];
    }
  })(opts.allowMethods);
  return /* @__PURE__ */ __name(async function cors2(c, next) {
    function set(key, value) {
      c.res.headers.set(key, value);
    }
    __name(set, "set");
    const allowOrigin = await findAllowOrigin(c.req.header("origin") || "", c);
    if (allowOrigin) {
      set("Access-Control-Allow-Origin", allowOrigin);
    }
    if (opts.credentials) {
      set("Access-Control-Allow-Credentials", "true");
    }
    if (opts.exposeHeaders?.length) {
      set("Access-Control-Expose-Headers", opts.exposeHeaders.join(","));
    }
    if (c.req.method === "OPTIONS") {
      if (opts.origin !== "*") {
        set("Vary", "Origin");
      }
      if (opts.maxAge != null) {
        set("Access-Control-Max-Age", opts.maxAge.toString());
      }
      const allowMethods = await findAllowMethods(c.req.header("origin") || "", c);
      if (allowMethods.length) {
        set("Access-Control-Allow-Methods", allowMethods.join(","));
      }
      let headers = opts.allowHeaders;
      if (!headers?.length) {
        const requestHeaders = c.req.header("Access-Control-Request-Headers");
        if (requestHeaders) {
          headers = requestHeaders.split(/\s*,\s*/);
        }
      }
      if (headers?.length) {
        set("Access-Control-Allow-Headers", headers.join(","));
        c.res.headers.append("Vary", "Access-Control-Request-Headers");
      }
      c.res.headers.delete("Content-Length");
      c.res.headers.delete("Content-Type");
      return new Response(null, {
        headers: c.res.headers,
        status: 204,
        statusText: "No Content"
      });
    }
    await next();
    if (opts.origin !== "*") {
      c.header("Vary", "Origin", { append: true });
    }
  }, "cors2");
}, "cors");

// node_modules/hono/dist/adapter/cloudflare-workers/index.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/adapter/cloudflare-workers/serve-static-module.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/adapter/cloudflare-workers/serve-static.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/middleware/serve-static/index.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/utils/compress.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/utils/mime.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/middleware/serve-static/path.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/middleware/serve-static/index.js
var ENCODINGS = {
  br: ".br",
  zstd: ".zst",
  gzip: ".gz"
};
var ENCODINGS_ORDERED_KEYS = Object.keys(ENCODINGS);

// node_modules/hono/dist/adapter/cloudflare-workers/utils.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/adapter/cloudflare-workers/websocket.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/hono/dist/helper/websocket/index.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var WSContext = class {
  static {
    __name(this, "WSContext");
  }
  #init;
  constructor(init) {
    this.#init = init;
    this.raw = init.raw;
    this.url = init.url ? new URL(init.url) : null;
    this.protocol = init.protocol ?? null;
  }
  send(source, options) {
    this.#init.send(source, options ?? {});
  }
  raw;
  binaryType = "arraybuffer";
  get readyState() {
    return this.#init.readyState;
  }
  url;
  protocol;
  close(code, reason) {
    this.#init.close(code, reason);
  }
};
var defineWebSocketHelper = /* @__PURE__ */ __name((handler) => {
  return ((...args) => {
    if (typeof args[0] === "function") {
      const [createEvents, options] = args;
      return /* @__PURE__ */ __name(async function upgradeWebSocket2(c, next) {
        const events = await createEvents(c);
        const result = await handler(c, events, options);
        if (result) {
          return result;
        }
        await next();
      }, "upgradeWebSocket");
    } else {
      const [c, events, options] = args;
      return (async () => {
        const upgraded = await handler(c, events, options);
        if (!upgraded) {
          throw new Error("Failed to upgrade WebSocket");
        }
        return upgraded;
      })();
    }
  });
}, "defineWebSocketHelper");

// node_modules/hono/dist/adapter/cloudflare-workers/websocket.js
var upgradeWebSocket = defineWebSocketHelper(async (c, events) => {
  const upgradeHeader = c.req.header("Upgrade");
  if (upgradeHeader !== "websocket") {
    return;
  }
  const webSocketPair = new WebSocketPair();
  const client = webSocketPair[0];
  const server = webSocketPair[1];
  const wsContext = new WSContext({
    close: /* @__PURE__ */ __name((code, reason) => server.close(code, reason), "close"),
    get protocol() {
      return server.protocol;
    },
    raw: server,
    get readyState() {
      return server.readyState;
    },
    url: server.url ? new URL(server.url) : null,
    send: /* @__PURE__ */ __name((source) => server.send(source), "send")
  });
  if (events.onClose) {
    server.addEventListener("close", (evt) => events.onClose?.(evt, wsContext));
  }
  if (events.onMessage) {
    server.addEventListener("message", (evt) => events.onMessage?.(evt, wsContext));
  }
  if (events.onError) {
    server.addEventListener("error", (evt) => events.onError?.(evt, wsContext));
  }
  server.accept?.();
  return new Response(null, {
    status: 101,
    // @ts-expect-error - webSocket is not typed
    webSocket: client
  });
});

// node_modules/hono/dist/adapter/cloudflare-workers/conninfo.js
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// src/lib/appwrite.ts
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/index.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/client.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-fetch-native-with-agent/dist/native.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var e = globalThis.Blob;
var o = globalThis.File;
var a = globalThis.FormData;
var s = globalThis.Headers;
var t = globalThis.Request;
var h = globalThis.Response;
var i = globalThis.AbortController;
var l = globalThis.fetch || (() => {
  throw new Error("[node-fetch-native] Failed to fetch: `globalThis.fetch` is not available!");
});

// node_modules/node-fetch-native-with-agent/dist/agent-stub.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var o2 = Object.defineProperty;
var e2 = /* @__PURE__ */ __name((t2, c) => o2(t2, "name", { value: c, configurable: true }), "e");
var i2 = Object.defineProperty;
var r = e2((t2, c) => i2(t2, "name", { value: c, configurable: true }), "e");
function a2() {
  return { agent: void 0, dispatcher: void 0 };
}
__name(a2, "a");
e2(a2, "createAgent"), r(a2, "createAgent");
function n() {
  return globalThis.fetch;
}
__name(n, "n");
e2(n, "createFetch"), r(n, "createFetch");
var h2 = globalThis.fetch;

// node_modules/node-appwrite/dist/client.mjs
var import_json_bigint2 = __toESM(require_json_bigint(), 1);

// node_modules/node-appwrite/dist/query.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var import_json_bigint = __toESM(require_json_bigint(), 1);
var JSONbig = (0, import_json_bigint.default)({ useNativeBigInt: true });
var _Query = class _Query2 {
  static {
    __name(this, "_Query");
  }
  /**
   * Constructor for Query class.
   *
   * @param {string} method
   * @param {AttributesTypes} attribute
   * @param {QueryTypes} values
   */
  constructor(method, attribute, values) {
    this.method = method;
    this.attribute = attribute;
    if (values !== void 0) {
      if (Array.isArray(values)) {
        this.values = values;
      } else {
        this.values = [values];
      }
    }
  }
  /**
   * Convert the query object to a JSON string.
   *
   * @returns {string}
   */
  toString() {
    return JSONbig.stringify({
      method: this.method,
      attribute: this.attribute,
      values: this.values
    });
  }
};
_Query.equal = (attribute, value) => new _Query("equal", attribute, value).toString();
_Query.notEqual = (attribute, value) => new _Query("notEqual", attribute, value).toString();
_Query.regex = (attribute, pattern) => new _Query("regex", attribute, pattern).toString();
_Query.lessThan = (attribute, value) => new _Query("lessThan", attribute, value).toString();
_Query.lessThanEqual = (attribute, value) => new _Query("lessThanEqual", attribute, value).toString();
_Query.greaterThan = (attribute, value) => new _Query("greaterThan", attribute, value).toString();
_Query.greaterThanEqual = (attribute, value) => new _Query("greaterThanEqual", attribute, value).toString();
_Query.isNull = (attribute) => new _Query("isNull", attribute).toString();
_Query.isNotNull = (attribute) => new _Query("isNotNull", attribute).toString();
_Query.exists = (attributes) => new _Query("exists", void 0, attributes).toString();
_Query.notExists = (attributes) => new _Query("notExists", void 0, attributes).toString();
_Query.between = (attribute, start, end) => new _Query("between", attribute, [start, end]).toString();
_Query.startsWith = (attribute, value) => new _Query("startsWith", attribute, value).toString();
_Query.endsWith = (attribute, value) => new _Query("endsWith", attribute, value).toString();
_Query.select = (attributes) => new _Query("select", void 0, attributes).toString();
_Query.search = (attribute, value) => new _Query("search", attribute, value).toString();
_Query.orderDesc = (attribute) => new _Query("orderDesc", attribute).toString();
_Query.orderAsc = (attribute) => new _Query("orderAsc", attribute).toString();
_Query.orderRandom = () => new _Query("orderRandom").toString();
_Query.cursorAfter = (documentId) => new _Query("cursorAfter", void 0, documentId).toString();
_Query.cursorBefore = (documentId) => new _Query("cursorBefore", void 0, documentId).toString();
_Query.limit = (limit) => new _Query("limit", void 0, limit).toString();
_Query.offset = (offset) => new _Query("offset", void 0, offset).toString();
_Query.contains = (attribute, value) => new _Query("contains", attribute, value).toString();
_Query.containsAny = (attribute, value) => new _Query("containsAny", attribute, value).toString();
_Query.containsAll = (attribute, value) => new _Query("containsAll", attribute, value).toString();
_Query.notContains = (attribute, value) => new _Query("notContains", attribute, value).toString();
_Query.notSearch = (attribute, value) => new _Query("notSearch", attribute, value).toString();
_Query.notBetween = (attribute, start, end) => new _Query("notBetween", attribute, [start, end]).toString();
_Query.notStartsWith = (attribute, value) => new _Query("notStartsWith", attribute, value).toString();
_Query.notEndsWith = (attribute, value) => new _Query("notEndsWith", attribute, value).toString();
_Query.createdBefore = (value) => _Query.lessThan("$createdAt", value);
_Query.createdAfter = (value) => _Query.greaterThan("$createdAt", value);
_Query.createdBetween = (start, end) => _Query.between("$createdAt", start, end);
_Query.updatedBefore = (value) => _Query.lessThan("$updatedAt", value);
_Query.updatedAfter = (value) => _Query.greaterThan("$updatedAt", value);
_Query.updatedBetween = (start, end) => _Query.between("$updatedAt", start, end);
_Query.or = (queries) => new _Query("or", void 0, queries.map((query) => JSONbig.parse(query))).toString();
_Query.and = (queries) => new _Query("and", void 0, queries.map((query) => JSONbig.parse(query))).toString();
_Query.elemMatch = (attribute, queries) => new _Query(
  "elemMatch",
  attribute,
  queries.map((query) => JSONbig.parse(query))
).toString();
_Query.distanceEqual = (attribute, values, distance, meters = true) => new _Query("distanceEqual", attribute, [[values, distance, meters]]).toString();
_Query.distanceNotEqual = (attribute, values, distance, meters = true) => new _Query("distanceNotEqual", attribute, [[values, distance, meters]]).toString();
_Query.distanceGreaterThan = (attribute, values, distance, meters = true) => new _Query("distanceGreaterThan", attribute, [[values, distance, meters]]).toString();
_Query.distanceLessThan = (attribute, values, distance, meters = true) => new _Query("distanceLessThan", attribute, [[values, distance, meters]]).toString();
_Query.intersects = (attribute, values) => new _Query("intersects", attribute, [values]).toString();
_Query.notIntersects = (attribute, values) => new _Query("notIntersects", attribute, [values]).toString();
_Query.crosses = (attribute, values) => new _Query("crosses", attribute, [values]).toString();
_Query.notCrosses = (attribute, values) => new _Query("notCrosses", attribute, [values]).toString();
_Query.overlaps = (attribute, values) => new _Query("overlaps", attribute, [values]).toString();
_Query.notOverlaps = (attribute, values) => new _Query("notOverlaps", attribute, [values]).toString();
_Query.touches = (attribute, values) => new _Query("touches", attribute, [values]).toString();
_Query.notTouches = (attribute, values) => new _Query("notTouches", attribute, [values]).toString();
var Query = _Query;

// node_modules/node-appwrite/dist/client.mjs
var JSONbigParser = (0, import_json_bigint2.default)({ storeAsString: false });
var JSONbigSerializer = (0, import_json_bigint2.default)({ useNativeBigInt: true });
var MAX_SAFE = BigInt(Number.MAX_SAFE_INTEGER);
var MIN_SAFE = BigInt(Number.MIN_SAFE_INTEGER);
var MAX_INT64 = BigInt("9223372036854775807");
var MIN_INT64 = BigInt("-9223372036854775808");
function isBigNumber(value) {
  return value !== null && typeof value === "object" && value._isBigNumber === true && typeof value.isInteger === "function" && typeof value.toFixed === "function" && typeof value.toNumber === "function";
}
__name(isBigNumber, "isBigNumber");
function reviver(_key, value) {
  if (isBigNumber(value)) {
    if (value.isInteger()) {
      const str = value.toFixed();
      const bi = BigInt(str);
      if (bi >= MIN_SAFE && bi <= MAX_SAFE) {
        return Number(str);
      }
      if (bi >= MIN_INT64 && bi <= MAX_INT64) {
        return bi;
      }
      return value.toNumber();
    }
    return value.toNumber();
  }
  return value;
}
__name(reviver, "reviver");
var JSONbig2 = {
  parse: /* @__PURE__ */ __name((text) => JSONbigParser.parse(text, reviver), "parse"),
  stringify: JSONbigSerializer.stringify
};
var AppwriteException = class extends Error {
  static {
    __name(this, "AppwriteException");
  }
  constructor(message, code = 0, type = "", response = "") {
    super(message);
    this.name = "AppwriteException";
    this.message = message;
    this.code = code;
    this.type = type;
    this.response = response;
  }
};
function getUserAgent() {
  let ua = "AppwriteNodeJSSDK/22.1.3";
  const platform2 = [];
  if (typeof process !== "undefined") {
    if (typeof process.platform === "string")
      platform2.push(process.platform);
    if (typeof process.arch === "string")
      platform2.push(process.arch);
  }
  if (platform2.length > 0) {
    ua += ` (${platform2.join("; ")})`;
  }
  if (typeof navigator !== "undefined" && true) {
    ua += ` ${"Cloudflare-Workers"}`;
  } else if (typeof globalThis.EdgeRuntime === "string") {
    ua += ` EdgeRuntime`;
  } else if (typeof process !== "undefined" && typeof process.version === "string") {
    ua += ` Node.js/${process.version}`;
  }
  return ua;
}
__name(getUserAgent, "getUserAgent");
var _Client = class _Client2 {
  static {
    __name(this, "_Client");
  }
  constructor() {
    this.config = {
      endpoint: "https://cloud.appwrite.io/v1",
      selfSigned: false,
      project: "",
      key: "",
      jwt: "",
      locale: "",
      session: "",
      forwardeduseragent: ""
    };
    this.headers = {
      "x-sdk-name": "Node.js",
      "x-sdk-platform": "server",
      "x-sdk-language": "nodejs",
      "x-sdk-version": "22.1.3",
      "user-agent": getUserAgent(),
      "X-Appwrite-Response-Format": "1.8.0"
    };
  }
  /**
   * Set Endpoint
   *
   * Your project endpoint
   *
   * @param {string} endpoint
   *
   * @returns {this}
   */
  setEndpoint(endpoint) {
    if (!endpoint || typeof endpoint !== "string") {
      throw new AppwriteException("Endpoint must be a valid string");
    }
    if (!endpoint.startsWith("http://") && !endpoint.startsWith("https://")) {
      throw new AppwriteException("Invalid endpoint URL: " + endpoint);
    }
    this.config.endpoint = endpoint;
    return this;
  }
  /**
   * Set self-signed
   *
   * @param {boolean} selfSigned
   *
   * @returns {this}
   */
  setSelfSigned(selfSigned) {
    if (typeof globalThis.EdgeRuntime !== "undefined") {
      console.warn("setSelfSigned is not supported in edge runtimes.");
    }
    this.config.selfSigned = selfSigned;
    return this;
  }
  /**
   * Add header
   *
   * @param {string} header
   * @param {string} value
   *
   * @returns {this}
   */
  addHeader(header, value) {
    this.headers[header.toLowerCase()] = value;
    return this;
  }
  /**
   * Set Project
   *
   * Your project ID
   *
   * @param value string
   *
   * @return {this}
   */
  setProject(value) {
    this.headers["X-Appwrite-Project"] = value;
    this.config.project = value;
    return this;
  }
  /**
   * Set Key
   *
   * Your secret API key
   *
   * @param value string
   *
   * @return {this}
   */
  setKey(value) {
    this.headers["X-Appwrite-Key"] = value;
    this.config.key = value;
    return this;
  }
  /**
   * Set JWT
   *
   * Your secret JSON Web Token
   *
   * @param value string
   *
   * @return {this}
   */
  setJWT(value) {
    this.headers["X-Appwrite-JWT"] = value;
    this.config.jwt = value;
    return this;
  }
  /**
   * Set Locale
   *
   * @param value string
   *
   * @return {this}
   */
  setLocale(value) {
    this.headers["X-Appwrite-Locale"] = value;
    this.config.locale = value;
    return this;
  }
  /**
   * Set Session
   *
   * The user session to authenticate with
   *
   * @param value string
   *
   * @return {this}
   */
  setSession(value) {
    this.headers["X-Appwrite-Session"] = value;
    this.config.session = value;
    return this;
  }
  /**
   * Set ForwardedUserAgent
   *
   * The user agent string of the client that made the request
   *
   * @param value string
   *
   * @return {this}
   */
  setForwardedUserAgent(value) {
    this.headers["X-Forwarded-User-Agent"] = value;
    this.config.forwardeduseragent = value;
    return this;
  }
  prepareRequest(method, url, headers = {}, params = {}) {
    method = method.toUpperCase();
    headers = Object.assign({}, this.headers, headers);
    let options = {
      method,
      headers,
      ...a2(this.config.endpoint, { rejectUnauthorized: !this.config.selfSigned })
    };
    if (method === "GET") {
      for (const [key, value] of Object.entries(_Client2.flatten(params))) {
        url.searchParams.append(key, value);
      }
    } else {
      switch (headers["content-type"]) {
        case "application/json":
          options.body = JSONbig2.stringify(params);
          break;
        case "multipart/form-data":
          const formData = new a();
          for (const [key, value] of Object.entries(params)) {
            if (value instanceof o) {
              formData.append(key, value, value.name);
            } else if (Array.isArray(value)) {
              for (const nestedValue of value) {
                formData.append(`${key}[]`, nestedValue);
              }
            } else {
              formData.append(key, value);
            }
          }
          options.body = formData;
          delete headers["content-type"];
          break;
      }
    }
    return { uri: url.toString(), options };
  }
  async chunkedUpload(method, url, headers = {}, originalPayload = {}, onProgress) {
    const [fileParam, file] = Object.entries(originalPayload).find(([_, value]) => value instanceof o) ?? [];
    if (!file || !fileParam) {
      throw new Error("File not found in payload");
    }
    if (file.size <= _Client2.CHUNK_SIZE) {
      return await this.call(method, url, headers, originalPayload);
    }
    let start = 0;
    let response = null;
    while (start < file.size) {
      let end = start + _Client2.CHUNK_SIZE;
      if (end >= file.size) {
        end = file.size;
      }
      headers["content-range"] = `bytes ${start}-${end - 1}/${file.size}`;
      const chunk = file.slice(start, end);
      let payload = { ...originalPayload };
      payload[fileParam] = new o([chunk], file.name);
      response = await this.call(method, url, headers, payload);
      if (onProgress && typeof onProgress === "function") {
        onProgress({
          $id: response.$id,
          progress: Math.round(end / file.size * 100),
          sizeUploaded: end,
          chunksTotal: Math.ceil(file.size / _Client2.CHUNK_SIZE),
          chunksUploaded: Math.ceil(end / _Client2.CHUNK_SIZE)
        });
      }
      if (response && response.$id) {
        headers["x-appwrite-id"] = response.$id;
      }
      start = end;
    }
    return response;
  }
  async ping() {
    return this.call("GET", new URL(this.config.endpoint + "/ping"));
  }
  async redirect(method, url, headers = {}, params = {}) {
    const { uri, options } = this.prepareRequest(method, url, headers, params);
    const response = await l(uri, {
      ...options,
      redirect: "manual"
    });
    if (response.status !== 301 && response.status !== 302) {
      throw new AppwriteException("Invalid redirect", response.status);
    }
    return response.headers.get("location") || "";
  }
  async call(method, url, headers = {}, params = {}, responseType = "json") {
    var _a, _b;
    const { uri, options } = this.prepareRequest(method, url, headers, params);
    let data = null;
    const response = await l(uri, options);
    const warnings = response.headers.get("x-appwrite-warning");
    if (warnings) {
      warnings.split(";").forEach((warning) => console.warn("Warning: " + warning));
    }
    if ((_a = response.headers.get("content-type")) == null ? void 0 : _a.includes("application/json")) {
      data = JSONbig2.parse(await response.text());
    } else if (responseType === "arrayBuffer") {
      data = await response.arrayBuffer();
    } else {
      data = {
        message: await response.text()
      };
    }
    if (400 <= response.status) {
      let responseText = "";
      if (((_b = response.headers.get("content-type")) == null ? void 0 : _b.includes("application/json")) || responseType === "arrayBuffer") {
        responseText = JSONbig2.stringify(data);
      } else {
        responseText = data == null ? void 0 : data.message;
      }
      throw new AppwriteException(data == null ? void 0 : data.message, response.status, data == null ? void 0 : data.type, responseText);
    }
    return data;
  }
  static flatten(data, prefix = "") {
    let output = {};
    for (const [key, value] of Object.entries(data)) {
      let finalKey = prefix ? prefix + "[" + key + "]" : key;
      if (Array.isArray(value)) {
        output = { ...output, ..._Client2.flatten(value, finalKey) };
      } else {
        output[finalKey] = value;
      }
    }
    return output;
  }
};
_Client.CHUNK_SIZE = 1024 * 1024 * 5;
var Client = _Client;

// node_modules/node-appwrite/dist/services/account.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/activities.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/avatars.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/backups.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/databases.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var Databases = class {
  static {
    __name(this, "Databases");
  }
  constructor(client) {
    this.client = client;
  }
  list(paramsOrFirst, ...rest) {
    let params;
    if (!paramsOrFirst || paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        queries: paramsOrFirst,
        search: rest[0],
        total: rest[1]
      };
    }
    const queries = params.queries;
    const search = params.search;
    const total = params.total;
    const apiPath = "/databases";
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof search !== "undefined") {
      payload["search"] = search;
    }
    if (typeof total !== "undefined") {
      payload["total"] = total;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  create(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        name: rest[0],
        enabled: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const name = params.name;
    const enabled = params.enabled;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof name === "undefined") {
      throw new AppwriteException('Missing required parameter: "name"');
    }
    const apiPath = "/databases";
    const payload = {};
    if (typeof databaseId !== "undefined") {
      payload["databaseId"] = databaseId;
    }
    if (typeof name !== "undefined") {
      payload["name"] = name;
    }
    if (typeof enabled !== "undefined") {
      payload["enabled"] = enabled;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  listTransactions(paramsOrFirst) {
    let params;
    if (!paramsOrFirst || paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        queries: paramsOrFirst
      };
    }
    const queries = params.queries;
    const apiPath = "/databases/transactions";
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  createTransaction(paramsOrFirst) {
    let params;
    if (!paramsOrFirst || paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        ttl: paramsOrFirst
      };
    }
    const ttl = params.ttl;
    const apiPath = "/databases/transactions";
    const payload = {};
    if (typeof ttl !== "undefined") {
      payload["ttl"] = ttl;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  getTransaction(paramsOrFirst) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        transactionId: paramsOrFirst
      };
    }
    const transactionId = params.transactionId;
    if (typeof transactionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "transactionId"');
    }
    const apiPath = "/databases/transactions/{transactionId}".replace("{transactionId}", transactionId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  updateTransaction(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        transactionId: paramsOrFirst,
        commit: rest[0],
        rollback: rest[1]
      };
    }
    const transactionId = params.transactionId;
    const commit = params.commit;
    const rollback = params.rollback;
    if (typeof transactionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "transactionId"');
    }
    const apiPath = "/databases/transactions/{transactionId}".replace("{transactionId}", transactionId);
    const payload = {};
    if (typeof commit !== "undefined") {
      payload["commit"] = commit;
    }
    if (typeof rollback !== "undefined") {
      payload["rollback"] = rollback;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteTransaction(paramsOrFirst) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        transactionId: paramsOrFirst
      };
    }
    const transactionId = params.transactionId;
    if (typeof transactionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "transactionId"');
    }
    const apiPath = "/databases/transactions/{transactionId}".replace("{transactionId}", transactionId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  createOperations(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        transactionId: paramsOrFirst,
        operations: rest[0]
      };
    }
    const transactionId = params.transactionId;
    const operations = params.operations;
    if (typeof transactionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "transactionId"');
    }
    const apiPath = "/databases/transactions/{transactionId}/operations".replace("{transactionId}", transactionId);
    const payload = {};
    if (typeof operations !== "undefined") {
      payload["operations"] = operations;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  get(paramsOrFirst) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst
      };
    }
    const databaseId = params.databaseId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    const apiPath = "/databases/{databaseId}".replace("{databaseId}", databaseId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  update(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        name: rest[0],
        enabled: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const name = params.name;
    const enabled = params.enabled;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    const apiPath = "/databases/{databaseId}".replace("{databaseId}", databaseId);
    const payload = {};
    if (typeof name !== "undefined") {
      payload["name"] = name;
    }
    if (typeof enabled !== "undefined") {
      payload["enabled"] = enabled;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "put",
      uri,
      apiHeaders,
      payload
    );
  }
  delete(paramsOrFirst) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst
      };
    }
    const databaseId = params.databaseId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    const apiPath = "/databases/{databaseId}".replace("{databaseId}", databaseId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  listCollections(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        queries: rest[0],
        search: rest[1],
        total: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const queries = params.queries;
    const search = params.search;
    const total = params.total;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    const apiPath = "/databases/{databaseId}/collections".replace("{databaseId}", databaseId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof search !== "undefined") {
      payload["search"] = search;
    }
    if (typeof total !== "undefined") {
      payload["total"] = total;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  createCollection(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        name: rest[1],
        permissions: rest[2],
        documentSecurity: rest[3],
        enabled: rest[4],
        attributes: rest[5],
        indexes: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const name = params.name;
    const permissions = params.permissions;
    const documentSecurity = params.documentSecurity;
    const enabled = params.enabled;
    const attributes = params.attributes;
    const indexes = params.indexes;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof name === "undefined") {
      throw new AppwriteException('Missing required parameter: "name"');
    }
    const apiPath = "/databases/{databaseId}/collections".replace("{databaseId}", databaseId);
    const payload = {};
    if (typeof collectionId !== "undefined") {
      payload["collectionId"] = collectionId;
    }
    if (typeof name !== "undefined") {
      payload["name"] = name;
    }
    if (typeof permissions !== "undefined") {
      payload["permissions"] = permissions;
    }
    if (typeof documentSecurity !== "undefined") {
      payload["documentSecurity"] = documentSecurity;
    }
    if (typeof enabled !== "undefined") {
      payload["enabled"] = enabled;
    }
    if (typeof attributes !== "undefined") {
      payload["attributes"] = attributes;
    }
    if (typeof indexes !== "undefined") {
      payload["indexes"] = indexes;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  getCollection(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  updateCollection(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        name: rest[1],
        permissions: rest[2],
        documentSecurity: rest[3],
        enabled: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const name = params.name;
    const permissions = params.permissions;
    const documentSecurity = params.documentSecurity;
    const enabled = params.enabled;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof name !== "undefined") {
      payload["name"] = name;
    }
    if (typeof permissions !== "undefined") {
      payload["permissions"] = permissions;
    }
    if (typeof documentSecurity !== "undefined") {
      payload["documentSecurity"] = documentSecurity;
    }
    if (typeof enabled !== "undefined") {
      payload["enabled"] = enabled;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "put",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteCollection(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  listAttributes(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        queries: rest[1],
        total: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const queries = params.queries;
    const total = params.total;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof total !== "undefined") {
      payload["total"] = total;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  createBooleanAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/boolean".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateBooleanAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/boolean/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createDatetimeAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/datetime".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateDatetimeAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/datetime/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createEmailAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/email".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateEmailAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/email/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createEnumAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        elements: rest[2],
        required: rest[3],
        xdefault: rest[4],
        array: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const elements = params.elements;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof elements === "undefined") {
      throw new AppwriteException('Missing required parameter: "elements"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/enum".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof elements !== "undefined") {
      payload["elements"] = elements;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateEnumAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        elements: rest[2],
        required: rest[3],
        xdefault: rest[4],
        newKey: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const elements = params.elements;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof elements === "undefined") {
      throw new AppwriteException('Missing required parameter: "elements"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/enum/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof elements !== "undefined") {
      payload["elements"] = elements;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createFloatAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        min: rest[3],
        max: rest[4],
        xdefault: rest[5],
        array: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const min = params.min;
    const max = params.max;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/float".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof min !== "undefined") {
      payload["min"] = min;
    }
    if (typeof max !== "undefined") {
      payload["max"] = max;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateFloatAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        min: rest[4],
        max: rest[5],
        newKey: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const min = params.min;
    const max = params.max;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/float/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof min !== "undefined") {
      payload["min"] = min;
    }
    if (typeof max !== "undefined") {
      payload["max"] = max;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createIntegerAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        min: rest[3],
        max: rest[4],
        xdefault: rest[5],
        array: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const min = params.min;
    const max = params.max;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/integer".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof min !== "undefined") {
      payload["min"] = min;
    }
    if (typeof max !== "undefined") {
      payload["max"] = max;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateIntegerAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        min: rest[4],
        max: rest[5],
        newKey: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const min = params.min;
    const max = params.max;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/integer/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof min !== "undefined") {
      payload["min"] = min;
    }
    if (typeof max !== "undefined") {
      payload["max"] = max;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createIpAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/ip".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateIpAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/ip/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createLineAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/line".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateLineAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/line/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createLongtextAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4],
        encrypt: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    const encrypt = params.encrypt;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/longtext".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    if (typeof encrypt !== "undefined") {
      payload["encrypt"] = encrypt;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateLongtextAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/longtext/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createMediumtextAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4],
        encrypt: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    const encrypt = params.encrypt;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/mediumtext".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    if (typeof encrypt !== "undefined") {
      payload["encrypt"] = encrypt;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateMediumtextAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/mediumtext/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createPointAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/point".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updatePointAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/point/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createPolygonAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/polygon".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updatePolygonAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/polygon/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createRelationshipAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        relatedCollectionId: rest[1],
        type: rest[2],
        twoWay: rest[3],
        key: rest[4],
        twoWayKey: rest[5],
        onDelete: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const relatedCollectionId = params.relatedCollectionId;
    const type = params.type;
    const twoWay = params.twoWay;
    const key = params.key;
    const twoWayKey = params.twoWayKey;
    const onDelete = params.onDelete;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof relatedCollectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "relatedCollectionId"');
    }
    if (typeof type === "undefined") {
      throw new AppwriteException('Missing required parameter: "type"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/relationship".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof relatedCollectionId !== "undefined") {
      payload["relatedCollectionId"] = relatedCollectionId;
    }
    if (typeof type !== "undefined") {
      payload["type"] = type;
    }
    if (typeof twoWay !== "undefined") {
      payload["twoWay"] = twoWay;
    }
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof twoWayKey !== "undefined") {
      payload["twoWayKey"] = twoWayKey;
    }
    if (typeof onDelete !== "undefined") {
      payload["onDelete"] = onDelete;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateRelationshipAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        onDelete: rest[2],
        newKey: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const onDelete = params.onDelete;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/relationship/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof onDelete !== "undefined") {
      payload["onDelete"] = onDelete;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createStringAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        size: rest[2],
        required: rest[3],
        xdefault: rest[4],
        array: rest[5],
        encrypt: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const size = params.size;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    const encrypt = params.encrypt;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof size === "undefined") {
      throw new AppwriteException('Missing required parameter: "size"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/string".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof size !== "undefined") {
      payload["size"] = size;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    if (typeof encrypt !== "undefined") {
      payload["encrypt"] = encrypt;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateStringAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        size: rest[4],
        newKey: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const size = params.size;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/string/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof size !== "undefined") {
      payload["size"] = size;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createTextAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4],
        encrypt: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    const encrypt = params.encrypt;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/text".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    if (typeof encrypt !== "undefined") {
      payload["encrypt"] = encrypt;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateTextAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/text/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createUrlAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/url".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateUrlAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/url/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createVarcharAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        size: rest[2],
        required: rest[3],
        xdefault: rest[4],
        array: rest[5],
        encrypt: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const size = params.size;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    const encrypt = params.encrypt;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof size === "undefined") {
      throw new AppwriteException('Missing required parameter: "size"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/varchar".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof size !== "undefined") {
      payload["size"] = size;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    if (typeof encrypt !== "undefined") {
      payload["encrypt"] = encrypt;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateVarcharAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        size: rest[4],
        newKey: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const size = params.size;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/varchar/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof size !== "undefined") {
      payload["size"] = size;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  getAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  listDocuments(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        queries: rest[1],
        transactionId: rest[2],
        total: rest[3],
        ttl: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const queries = params.queries;
    const transactionId = params.transactionId;
    const total = params.total;
    const ttl = params.ttl;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    if (typeof total !== "undefined") {
      payload["total"] = total;
    }
    if (typeof ttl !== "undefined") {
      payload["ttl"] = ttl;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  createDocument(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        data: rest[2],
        permissions: rest[3],
        transactionId: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const data = params.data;
    const permissions = params.permissions;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    if (typeof data === "undefined") {
      throw new AppwriteException('Missing required parameter: "data"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof documentId !== "undefined") {
      payload["documentId"] = documentId;
    }
    if (typeof data !== "undefined") {
      payload["data"] = data;
    }
    if (typeof permissions !== "undefined") {
      payload["permissions"] = permissions;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  createDocuments(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documents: rest[1],
        transactionId: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documents = params.documents;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documents === "undefined") {
      throw new AppwriteException('Missing required parameter: "documents"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof documents !== "undefined") {
      payload["documents"] = documents;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  upsertDocuments(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documents: rest[1],
        transactionId: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documents = params.documents;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documents === "undefined") {
      throw new AppwriteException('Missing required parameter: "documents"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof documents !== "undefined") {
      payload["documents"] = documents;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "put",
      uri,
      apiHeaders,
      payload
    );
  }
  updateDocuments(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        data: rest[1],
        queries: rest[2],
        transactionId: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const data = params.data;
    const queries = params.queries;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof data !== "undefined") {
      payload["data"] = data;
    }
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteDocuments(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        queries: rest[1],
        transactionId: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const queries = params.queries;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  getDocument(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        queries: rest[2],
        transactionId: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const queries = params.queries;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  upsertDocument(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        data: rest[2],
        permissions: rest[3],
        transactionId: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const data = params.data;
    const permissions = params.permissions;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId);
    const payload = {};
    if (typeof data !== "undefined") {
      payload["data"] = data;
    }
    if (typeof permissions !== "undefined") {
      payload["permissions"] = permissions;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "put",
      uri,
      apiHeaders,
      payload
    );
  }
  updateDocument(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        data: rest[2],
        permissions: rest[3],
        transactionId: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const data = params.data;
    const permissions = params.permissions;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId);
    const payload = {};
    if (typeof data !== "undefined") {
      payload["data"] = data;
    }
    if (typeof permissions !== "undefined") {
      payload["permissions"] = permissions;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteDocument(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        transactionId: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId);
    const payload = {};
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  decrementDocumentAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        attribute: rest[2],
        value: rest[3],
        min: rest[4],
        transactionId: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const attribute = params.attribute;
    const value = params.value;
    const min = params.min;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    if (typeof attribute === "undefined") {
      throw new AppwriteException('Missing required parameter: "attribute"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}/{attribute}/decrement".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId).replace("{attribute}", attribute);
    const payload = {};
    if (typeof value !== "undefined") {
      payload["value"] = value;
    }
    if (typeof min !== "undefined") {
      payload["min"] = min;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  incrementDocumentAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        attribute: rest[2],
        value: rest[3],
        max: rest[4],
        transactionId: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const attribute = params.attribute;
    const value = params.value;
    const max = params.max;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    if (typeof attribute === "undefined") {
      throw new AppwriteException('Missing required parameter: "attribute"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}/{attribute}/increment".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId).replace("{attribute}", attribute);
    const payload = {};
    if (typeof value !== "undefined") {
      payload["value"] = value;
    }
    if (typeof max !== "undefined") {
      payload["max"] = max;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  listIndexes(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        queries: rest[1],
        total: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const queries = params.queries;
    const total = params.total;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/indexes".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof total !== "undefined") {
      payload["total"] = total;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  createIndex(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        type: rest[2],
        attributes: rest[3],
        orders: rest[4],
        lengths: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const type = params.type;
    const attributes = params.attributes;
    const orders = params.orders;
    const lengths = params.lengths;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof type === "undefined") {
      throw new AppwriteException('Missing required parameter: "type"');
    }
    if (typeof attributes === "undefined") {
      throw new AppwriteException('Missing required parameter: "attributes"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/indexes".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof type !== "undefined") {
      payload["type"] = type;
    }
    if (typeof attributes !== "undefined") {
      payload["attributes"] = attributes;
    }
    if (typeof orders !== "undefined") {
      payload["orders"] = orders;
    }
    if (typeof lengths !== "undefined") {
      payload["lengths"] = lengths;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  getIndex(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/indexes/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteIndex(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/indexes/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
};

// node_modules/node-appwrite/dist/services/functions.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/graphql.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/health.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/locale.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/messaging.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/sites.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/storage.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/tables-db.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/teams.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/tokens.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/services/users.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/permission.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var Permission = class {
  static {
    __name(this, "Permission");
  }
};
Permission.read = (role) => {
  return `read("${role}")`;
};
Permission.write = (role) => {
  return `write("${role}")`;
};
Permission.create = (role) => {
  return `create("${role}")`;
};
Permission.update = (role) => {
  return `update("${role}")`;
};
Permission.delete = (role) => {
  return `delete("${role}")`;
};

// node_modules/node-appwrite/dist/role.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/id.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/operator.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var _Operator = class _Operator2 {
  static {
    __name(this, "_Operator");
  }
  /**
   * Constructor for Operator class.
   *
   * @param {string} method
   * @param {OperatorValues} values
   */
  constructor(method, values) {
    this.method = method;
    if (values !== void 0) {
      if (Array.isArray(values)) {
        this.values = values;
      } else {
        this.values = [values];
      }
    }
  }
  /**
   * Convert the operator object to a JSON string.
   *
   * @returns {string}
   */
  toString() {
    return JSON.stringify({
      method: this.method,
      values: this.values
    });
  }
};
_Operator.increment = (value = 1, max) => {
  if (isNaN(value) || !isFinite(value)) {
    throw new Error("Value cannot be NaN or Infinity");
  }
  if (max !== void 0 && (isNaN(max) || !isFinite(max))) {
    throw new Error("Max cannot be NaN or Infinity");
  }
  const values = [value];
  if (max !== void 0) {
    values.push(max);
  }
  return new _Operator("increment", values).toString();
};
_Operator.decrement = (value = 1, min) => {
  if (isNaN(value) || !isFinite(value)) {
    throw new Error("Value cannot be NaN or Infinity");
  }
  if (min !== void 0 && (isNaN(min) || !isFinite(min))) {
    throw new Error("Min cannot be NaN or Infinity");
  }
  const values = [value];
  if (min !== void 0) {
    values.push(min);
  }
  return new _Operator("decrement", values).toString();
};
_Operator.multiply = (factor, max) => {
  if (isNaN(factor) || !isFinite(factor)) {
    throw new Error("Factor cannot be NaN or Infinity");
  }
  if (max !== void 0 && (isNaN(max) || !isFinite(max))) {
    throw new Error("Max cannot be NaN or Infinity");
  }
  const values = [factor];
  if (max !== void 0) {
    values.push(max);
  }
  return new _Operator("multiply", values).toString();
};
_Operator.divide = (divisor, min) => {
  if (isNaN(divisor) || !isFinite(divisor)) {
    throw new Error("Divisor cannot be NaN or Infinity");
  }
  if (min !== void 0 && (isNaN(min) || !isFinite(min))) {
    throw new Error("Min cannot be NaN or Infinity");
  }
  if (divisor === 0) {
    throw new Error("Divisor cannot be zero");
  }
  const values = [divisor];
  if (min !== void 0) {
    values.push(min);
  }
  return new _Operator("divide", values).toString();
};
_Operator.modulo = (divisor) => {
  if (isNaN(divisor) || !isFinite(divisor)) {
    throw new Error("Divisor cannot be NaN or Infinity");
  }
  if (divisor === 0) {
    throw new Error("Divisor cannot be zero");
  }
  return new _Operator("modulo", [divisor]).toString();
};
_Operator.power = (exponent, max) => {
  if (isNaN(exponent) || !isFinite(exponent)) {
    throw new Error("Exponent cannot be NaN or Infinity");
  }
  if (max !== void 0 && (isNaN(max) || !isFinite(max))) {
    throw new Error("Max cannot be NaN or Infinity");
  }
  const values = [exponent];
  if (max !== void 0) {
    values.push(max);
  }
  return new _Operator("power", values).toString();
};
_Operator.arrayAppend = (values) => new _Operator("arrayAppend", values).toString();
_Operator.arrayPrepend = (values) => new _Operator("arrayPrepend", values).toString();
_Operator.arrayInsert = (index, value) => new _Operator("arrayInsert", [index, value]).toString();
_Operator.arrayRemove = (value) => new _Operator("arrayRemove", [value]).toString();
_Operator.arrayUnique = () => new _Operator("arrayUnique", []).toString();
_Operator.arrayIntersect = (values) => new _Operator("arrayIntersect", values).toString();
_Operator.arrayDiff = (values) => new _Operator("arrayDiff", values).toString();
_Operator.arrayFilter = (condition, value) => {
  const values = [condition, value === void 0 ? null : value];
  return new _Operator("arrayFilter", values).toString();
};
_Operator.stringConcat = (value) => new _Operator("stringConcat", [value]).toString();
_Operator.stringReplace = (search, replace) => new _Operator("stringReplace", [search, replace]).toString();
_Operator.toggle = () => new _Operator("toggle", []).toString();
_Operator.dateAddDays = (days) => new _Operator("dateAddDays", [days]).toString();
_Operator.dateSubDays = (days) => new _Operator("dateSubDays", [days]).toString();
_Operator.dateSetNow = () => new _Operator("dateSetNow", []).toString();

// node_modules/node-appwrite/dist/enums/authenticator-type.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/authentication-factor.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/o-auth-provider.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/browser.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/credit-card.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/flag.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/theme.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/timezone.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/browser-permission.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/image-format.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/backup-services.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/relationship-type.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/relation-mutate.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/index-type.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/order-by.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/runtime.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/scopes.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/template-reference-type.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/vcs-reference-type.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/deployment-download-type.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/execution-method.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/name.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/message-priority.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/smtp-encryption.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/framework.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/build-runtime.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/adapter.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/compression.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/image-gravity.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/password-hash.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/messaging-provider-type.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/database-type.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/attribute-status.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/column-status.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/index-status.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/deployment-status.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/execution-trigger.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/execution-status.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/health-antivirus-status.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/health-check-status.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// node_modules/node-appwrite/dist/enums/message-status.mjs
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();

// src/lib/appwrite.ts
function toCents(amount) {
  return Math.round(amount * 100);
}
__name(toCents, "toCents");
var AppwriteService = class {
  static {
    __name(this, "AppwriteService");
  }
  constructor(env2) {
    this.env = env2;
    this.client = new Client().setEndpoint(env2.APPWRITE_ENDPOINT).setProject(env2.APPWRITE_PROJECT_ID).setKey(env2.APPWRITE_API_KEY);
    this.databases = new Databases(this.client);
  }
  async createTicket(ticketId, amount) {
    try {
      return await this.databases.createDocument(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        ticketId,
        {
          ticketId,
          amount,
          status: "pending"
        }
      );
    } catch (error3) {
      console.error("Appwrite createTicket error:", error3);
      throw error3;
    }
  }
  /**
   * Mark a ticket as paid, optionally persisting the sender name, RRN,
   * UPI ID, and the timestamp at which payment was confirmed.
   */
  async markAsPaid(ticketId, senderName, rrn, upiId) {
    try {
      if (rrn) {
        const existingWithRrn = await this.databases.listDocuments(
          this.env.APPWRITE_DATABASE_ID,
          this.env.APPWRITE_COLLECTION_ID,
          [Query.equal("rrn", rrn)]
        );
        if (existingWithRrn.total > 0) {
          console.log(`Duplicate RRN detected: ${rrn}. Skipping update.`);
          return null;
        }
      }
      const updatePayload = {
        status: "paid",
        paidAt: (/* @__PURE__ */ new Date()).toISOString()
      };
      if (senderName) {
        updatePayload.senderName = senderName;
      }
      if (rrn) {
        updatePayload.rrn = rrn;
      }
      if (upiId) {
        updatePayload.upiId = upiId;
      }
      console.log("Sending update to Appwrite for ticket:", ticketId);
      console.log("Update Payload:", JSON.stringify(updatePayload, null, 2));
      const updatedDoc = await this.databases.updateDocument(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        ticketId,
        updatePayload
      );
      return {
        id: updatedDoc.$id,
        ticketId: updatedDoc.ticketId,
        status: updatedDoc.status,
        amount: updatedDoc.amount,
        senderName: updatedDoc.senderName,
        rrn: updatedDoc.rrn ?? null,
        paidAt: updatedDoc.paidAt ?? null,
        upiId: updatedDoc.upiId ?? null,
        createdAt: updatedDoc.$createdAt
      };
    } catch (error3) {
      console.error("Appwrite markAsPaid error:", error3);
      return null;
    }
  }
  async getTicketStatus(ticketId) {
    try {
      const doc = await this.databases.getDocument(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        ticketId
      ).catch(() => null);
      if (!doc) {
        return null;
      }
      let currentStatus = doc.status;
      if (currentStatus === "pending") {
        const ticketTime = new Date(doc.$createdAt).getTime();
        const FIVE_MIN_MS = 5 * 60 * 1e3;
        if (Date.now() - ticketTime > FIVE_MIN_MS) {
          currentStatus = "cancelled";
          this.databases.updateDocument(
            this.env.APPWRITE_DATABASE_ID,
            this.env.APPWRITE_COLLECTION_ID,
            doc.$id,
            { status: "cancelled" }
          ).catch((err) => console.error("Lazy cancel DB update error:", err));
        }
      }
      return {
        id: doc.$id,
        ticketId: doc.ticketId,
        status: currentStatus,
        amount: doc.amount,
        senderName: doc.senderName,
        rrn: doc.rrn ?? null,
        paidAt: doc.paidAt ?? null,
        upiId: doc.upiId ?? null,
        createdAt: doc.$createdAt
        // Appwrite built-in timestamp
      };
    } catch (error3) {
      console.error("Appwrite getTicketStatus error:", error3);
      return null;
    }
  }
  async listRecentPendingTickets(limit = 20) {
    try {
      const response = await this.databases.listDocuments(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        [
          Query.equal("status", "pending"),
          Query.orderDesc("$createdAt"),
          Query.limit(limit)
        ]
      );
      return response.documents.map((doc) => ({
        id: doc.$id,
        ticketId: doc.ticketId,
        status: doc.status,
        amount: doc.amount,
        createdAt: doc.$createdAt
      }));
    } catch (error3) {
      console.error("Appwrite listRecentPendingTickets error:", error3);
      return [];
    }
  }
  async listRecentTickets(limit = 20) {
    try {
      const response = await this.databases.listDocuments(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        [
          Query.orderDesc("$createdAt"),
          Query.limit(limit)
        ]
      );
      return response.documents.map((doc) => ({
        id: doc.$id,
        ticketId: doc.ticketId,
        status: doc.status,
        amount: doc.amount,
        createdAt: doc.$createdAt
      }));
    } catch (error3) {
      console.error("Appwrite listRecentTickets error:", error3);
      return [];
    }
  }
  async listAllTickets(options = {}) {
    const { limit = 500, statusFilter, dateFrom, dateTo } = options;
    try {
      const queries = [
        Query.orderDesc("$createdAt"),
        Query.limit(Math.min(limit, 500))
      ];
      if (statusFilter && ["pending", "paid", "cancelled"].includes(statusFilter)) {
        queries.push(Query.equal("status", statusFilter));
      }
      if (dateFrom) {
        queries.push(Query.greaterThanEqual("$createdAt", dateFrom));
      }
      if (dateTo) {
        queries.push(Query.lessThanEqual("$createdAt", dateTo));
      }
      const response = await this.databases.listDocuments(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        queries
      );
      return response.documents.filter((doc) => !doc.ticketId?.startsWith("lock_")).map((doc) => ({
        id: doc.$id,
        ticketId: doc.ticketId,
        amount: doc.amount,
        status: doc.status,
        createdAt: doc.$createdAt,
        senderName: doc.senderName ?? null,
        rrn: doc.rrn ?? null,
        paidAt: doc.paidAt ?? null,
        upiId: doc.upiId ?? null
      }));
    } catch (error3) {
      console.error("Appwrite listAllTickets error:", error3);
      return [];
    }
  }
  async updateTicket(ticketId, fields) {
    try {
      const payload = {};
      if (fields.status !== void 0) payload.status = fields.status;
      if (fields.senderName !== void 0) payload.senderName = fields.senderName;
      if (fields.rrn !== void 0) payload.rrn = fields.rrn;
      if (fields.upiId !== void 0) payload.upiId = fields.upiId;
      if (fields.amount !== void 0) payload.amount = fields.amount;
      if (fields.paidAt !== void 0) payload.paidAt = fields.paidAt;
      if (Object.keys(payload).length === 0) return null;
      const doc = await this.databases.updateDocument(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        ticketId,
        payload
      );
      return {
        id: doc.$id,
        ticketId: doc.ticketId,
        amount: doc.amount,
        status: doc.status,
        createdAt: doc.$createdAt,
        senderName: doc.senderName ?? null,
        rrn: doc.rrn ?? null,
        paidAt: doc.paidAt ?? null,
        upiId: doc.upiId ?? null
      };
    } catch (error3) {
      console.error("Appwrite updateTicket error:", error3);
      return null;
    }
  }
  async getPendingDecimalsForAmount(baseAmount) {
    try {
      const candidates = await this.listRecentPendingTickets(100);
      const now = Date.now();
      const FIVE_MIN_MS = 5 * 60 * 1e3;
      const decimalsAllocated = [];
      let cancellationsLeft = 5;
      for (const ticket of candidates) {
        const ticketTime = new Date(ticket.createdAt).getTime();
        if (now - ticketTime > FIVE_MIN_MS) {
          if (cancellationsLeft > 0) {
            cancellationsLeft--;
            if (ticket.ticketId.startsWith("lock_")) {
              this.databases.deleteDocument(
                this.env.APPWRITE_DATABASE_ID,
                this.env.APPWRITE_COLLECTION_ID,
                ticket.id
              ).catch(
                (err) => console.log(
                  "Lazy cancel batch error (ignorable):",
                  err.message
                )
              );
            } else {
              this.databases.updateDocument(
                this.env.APPWRITE_DATABASE_ID,
                this.env.APPWRITE_COLLECTION_ID,
                ticket.id,
                { status: "cancelled" }
              ).catch(
                (err) => console.log(
                  "Lazy cancel batch error (ignorable):",
                  err.message
                )
              );
            }
          }
          continue;
        }
        if (Math.floor(ticket.amount) === baseAmount) {
          const decPart = Math.round((ticket.amount - baseAmount) * 100);
          decimalsAllocated.push(decPart);
        }
      }
      return decimalsAllocated;
    } catch (error3) {
      console.error("Appwrite getPendingDecimalsForAmount error:", error3);
      return [];
    }
  }
  /**
   * Attempts to acquire an absolute, atomic database-level lock for a specific decimal.
   * Uses Appwrite's Document ID uniqueness constraint to guarantee no two concurrent requests
   * can claim the exact same decimal simultaneously.
   */
  async claimDatabaseLock(baseAmount, decimal) {
    const lockDocId = `lock_${baseAmount}_${decimal.toString().padStart(2, "0")}`;
    const finalAmount = baseAmount + decimal / 100;
    try {
      await this.databases.createDocument(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        lockDocId,
        {
          ticketId: lockDocId,
          amount: finalAmount,
          status: "pending"
        }
      );
      return true;
    } catch (error3) {
      if (error3.code === 409) {
        try {
          const existingLock = await this.databases.getDocument(
            this.env.APPWRITE_DATABASE_ID,
            this.env.APPWRITE_COLLECTION_ID,
            lockDocId
          );
          const lockTime = new Date(existingLock.$createdAt).getTime();
          if (Date.now() - lockTime > 5 * 60 * 1e3) {
            try {
              await this.releaseDatabaseLock(baseAmount, decimal);
            } catch (e3) {
            }
            return await this.claimDatabaseLock(baseAmount, decimal);
          }
        } catch (e3) {
        }
        return false;
      }
      throw error3;
    }
  }
  /**
   * Releases an atomic database-level lock so the decimal can be reused immediately.
   */
  async releaseDatabaseLock(baseAmount, decimal) {
    const lockDocId = `lock_${baseAmount}_${decimal.toString().padStart(2, "0")}`;
    try {
      await this.databases.deleteDocument(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        lockDocId
      );
    } catch (error3) {
    }
  }
};

// src/admin-html.ts
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var ADMIN_HTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Payment Admin</title>
  <style>
    :root {
      --bg-app: #000000;
      --bg-card: rgba(255,255,255,0.04);
      --bg-card-hover: rgba(255,255,255,0.07);
      --bg-input: rgba(255,255,255,0.06);
      --bg-input-focus: rgba(255,255,255,0.09);
      --bg-modal: rgba(12,12,12,0.98);
      --bg-header: rgba(0,0,0,0.85);
      --border-default: rgba(255,255,255,0.08);
      --border-bright: rgba(255,255,255,0.16);
      --border-focus: rgba(10,132,255,0.55);
      --text-primary: #f5f5f7;
      --text-secondary: rgba(235,235,245,0.60);
      --text-tertiary: rgba(235,235,245,0.30);
      --blue: #0a84ff;
      --green: #30d158;
      --orange: #ff9f0a;
      --red: #ff453a;
      --purple: #bf5af2;
      --teal: #40c8e0;
      --font-ui: -apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", system-ui, "Helvetica Neue", Arial, sans-serif;
      --font-mono: "SF Mono", ui-monospace, "Cascadia Code", "Fira Code", Menlo, Monaco, Consolas, monospace;
      --r-sm: 6px; --r-md: 10px; --r-lg: 14px; --r-xl: 20px; --r-2xl: 28px; --r-full: 9999px;
      --shadow-sm: 0 2px 8px rgba(0,0,0,0.30);
      --shadow-md: 0 8px 24px rgba(0,0,0,0.45);
      --shadow-lg: 0 20px 48px rgba(0,0,0,0.55);
      --shadow-xl: 0 32px 80px rgba(0,0,0,0.65);
      --ease-spring: cubic-bezier(0.34,1.56,0.64,1);
      --ease-out: cubic-bezier(0.16,1,0.3,1);
      --ease-default: cubic-bezier(0.25,0.46,0.45,0.94);
    }
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    * { scrollbar-width: thin; scrollbar-color: rgba(255,255,255,0.15) transparent; }
    ::-webkit-scrollbar { width: 5px; height: 5px; }
    ::-webkit-scrollbar-track { background: transparent; }
    ::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.13); border-radius: 3px; }
    ::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.22); }
    html, body { height: 100%; }
    body {
      font-family: var(--font-ui);
      color: var(--text-primary);
      background:
        radial-gradient(ellipse at 15% 40%, rgba(10,132,255,0.055) 0%, transparent 55%),
        radial-gradient(ellipse at 80% 20%, rgba(191,90,242,0.04) 0%, transparent 50%),
        radial-gradient(ellipse at 60% 85%, rgba(48,209,88,0.035) 0%, transparent 50%),
        #000000;
      min-height: 100vh;
      -webkit-font-smoothing: antialiased;
    }

    /* \u2500\u2500 LOGIN \u2500\u2500 */
    #login-screen {
      display: flex; align-items: center; justify-content: center;
      min-height: 100vh;
    }
    .login-card {
      width: 360px;
      background: rgba(20,20,22,0.92);
      backdrop-filter: blur(40px) saturate(180%);
      border: 1px solid var(--border-bright);
      border-top-color: rgba(255,255,255,0.22);
      border-radius: var(--r-2xl);
      padding: 40px 36px;
      box-shadow: var(--shadow-xl);
    }
    .login-logo { font-size: 28px; font-weight: 700; letter-spacing: -0.5px; margin-bottom: 6px; }
    .login-sub { color: var(--text-secondary); font-size: 14px; margin-bottom: 32px; }
    .login-label { font-size: 11px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 8px; }
    .login-input {
      width: 100%; padding: 12px 14px; font-size: 15px; font-family: var(--font-ui);
      background: var(--bg-input); border: 1px solid var(--border-default);
      border-radius: var(--r-md); color: var(--text-primary); outline: none;
      transition: border-color 0.2s, background 0.2s;
    }
    .login-input:focus { background: var(--bg-input-focus); border-color: var(--border-focus); }
    .login-btn {
      width: 100%; margin-top: 20px; padding: 13px; font-size: 15px; font-weight: 600;
      background: var(--blue); color: #fff; border: none; border-radius: var(--r-md);
      cursor: pointer; transition: opacity 0.15s, transform 0.15s;
    }
    .login-btn:hover { opacity: 0.88; }
    .login-btn:active { transform: scale(0.98); }
    .login-err { color: var(--red); font-size: 13px; margin-top: 10px; min-height: 18px; text-align: center; }
    @keyframes shake {
      0%,100% { transform: translateX(0); }
      15% { transform: translateX(-7px); }
      30% { transform: translateX(6px); }
      45% { transform: translateX(-5px); }
      60% { transform: translateX(4px); }
      75% { transform: translateX(-3px); }
    }
    .login-card.shake { animation: shake 0.45s ease; }

    /* \u2500\u2500 DASHBOARD \u2500\u2500 */
    #dashboard { display: flex; flex-direction: column; min-height: 100vh; }

    /* Header */
    header {
      position: sticky; top: 0; z-index: 100;
      background: var(--bg-header);
      backdrop-filter: blur(30px) saturate(160%);
      border-bottom: 1px solid var(--border-default);
      padding: 0 24px;
      display: flex; align-items: center; gap: 16px; height: 56px;
    }
    .header-logo { font-size: 16px; font-weight: 700; letter-spacing: -0.3px; margin-right: 8px; white-space: nowrap; }
    .nav-tabs { display: flex; gap: 2px; background: rgba(255,255,255,0.05); border-radius: var(--r-md); padding: 3px; }
    .nav-tab {
      padding: 5px 14px; font-size: 13px; font-weight: 500; border-radius: var(--r-sm);
      cursor: pointer; color: var(--text-secondary); transition: all 0.18s; border: none; background: none;
    }
    .nav-tab.active { background: rgba(255,255,255,0.12); color: var(--text-primary); }
    .nav-tab:hover:not(.active) { color: var(--text-primary); background: rgba(255,255,255,0.06); }
    .header-spacer { flex: 1; }
    .rt-dot {
      width: 8px; height: 8px; border-radius: 50%; transition: background 0.3s;
    }
    .rt-dot.connected { background: var(--green); box-shadow: 0 0 6px var(--green); }
    .rt-dot.disconnected { background: rgba(255,255,255,0.2); }
    .rt-label { font-size: 11px; color: var(--text-tertiary); margin-right: 4px; }
    .header-btn {
      padding: 6px 12px; font-size: 12px; font-weight: 500;
      background: rgba(255,255,255,0.07); border: 1px solid var(--border-default);
      border-radius: var(--r-sm); color: var(--text-secondary); cursor: pointer; transition: all 0.15s;
    }
    .header-btn:hover { color: var(--text-primary); background: rgba(255,255,255,0.11); }

    /* Main content */
    main { flex: 1; padding: 24px; max-width: 1400px; margin: 0 auto; width: 100%; }

    /* Stats */
    .stats-grid {
      display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 20px;
    }
    .stat-card {
      background: var(--bg-card); border: 1px solid var(--border-default);
      border-top-color: var(--border-bright); border-radius: var(--r-xl);
      padding: 18px 20px; cursor: pointer; transition: all 0.2s;
      position: relative; overflow: hidden;
    }
    .stat-card::before {
      content: ''; position: absolute; inset: 0; opacity: 0;
      transition: opacity 0.2s; border-radius: inherit;
    }
    .stat-card:hover { background: var(--bg-card-hover); transform: translateY(-1px); box-shadow: var(--shadow-md); }
    .stat-card.active-filter::before { opacity: 1; }
    .stat-card[data-key="all"].active-filter::before { background: rgba(10,132,255,0.08); }
    .stat-card[data-key="paid"].active-filter::before { background: rgba(48,209,88,0.08); }
    .stat-card[data-key="pending"].active-filter::before { background: rgba(255,159,10,0.08); }
    .stat-card[data-key="cancelled"].active-filter::before { background: rgba(255,69,58,0.08); }
    .stat-label { font-size: 11px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-tertiary); margin-bottom: 8px; }
    .stat-value { font-size: 32px; font-weight: 700; letter-spacing: -1px; font-feature-settings: "tnum"; }
    .stat-card[data-key="all"] .stat-value { color: var(--blue); }
    .stat-card[data-key="paid"] .stat-value { color: var(--green); }
    .stat-card[data-key="pending"] .stat-value { color: var(--orange); }
    .stat-card[data-key="cancelled"] .stat-value { color: var(--red); }

    /* Toolbar */
    .toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 16px; flex-wrap: wrap; }
    .search-wrap { position: relative; flex: 1; min-width: 200px; max-width: 360px; }
    .search-icon { position: absolute; left: 10px; top: 50%; transform: translateY(-50%); color: var(--text-tertiary); pointer-events: none; }
    .search-input {
      width: 100%; padding: 8px 12px 8px 34px; font-size: 14px; font-family: var(--font-ui);
      background: var(--bg-input); border: 1px solid var(--border-default); border-radius: var(--r-md);
      color: var(--text-primary); outline: none; transition: all 0.2s;
    }
    .search-input:focus { background: var(--bg-input-focus); border-color: var(--border-focus); }
    .filter-pills { display: flex; gap: 6px; flex-wrap: wrap; }
    .pill {
      padding: 5px 12px; font-size: 12px; font-weight: 500; border-radius: var(--r-full);
      border: 1px solid var(--border-default); background: transparent; color: var(--text-secondary);
      cursor: pointer; transition: all 0.16s;
    }
    .pill:hover { color: var(--text-primary); border-color: var(--border-bright); }
    .pill.active { background: rgba(10,132,255,0.15); border-color: rgba(10,132,255,0.45); color: var(--blue); }
    .pill.active[data-status="paid"] { background: rgba(48,209,88,0.12); border-color: rgba(48,209,88,0.4); color: var(--green); }
    .pill.active[data-status="pending"] { background: rgba(255,159,10,0.12); border-color: rgba(255,159,10,0.4); color: var(--orange); }
    .pill.active[data-status="cancelled"] { background: rgba(255,69,58,0.1); border-color: rgba(255,69,58,0.35); color: var(--red); }
    .date-range { display: flex; align-items: center; gap: 6px; }
    .date-input {
      padding: 6px 10px; font-size: 12px; font-family: var(--font-ui);
      background: var(--bg-input); border: 1px solid var(--border-default); border-radius: var(--r-md);
      color: var(--text-secondary); outline: none; cursor: pointer; transition: all 0.2s;
    }
    .date-input:focus { border-color: var(--border-focus); color: var(--text-primary); }
    .date-sep { color: var(--text-tertiary); font-size: 12px; }
    .date-clear { padding: 5px 10px; font-size: 11px; cursor: pointer; color: var(--text-tertiary); background: none; border: none; transition: color 0.15s; }
    .date-clear:hover { color: var(--red); }

    /* Table */
    .table-card {
      background: var(--bg-card); border: 1px solid var(--border-default);
      border-top-color: var(--border-bright); border-radius: var(--r-xl);
      overflow: hidden; box-shadow: var(--shadow-sm);
    }
    .table-scroll { overflow-x: auto; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    thead th {
      padding: 11px 14px; text-align: left; font-size: 11px; font-weight: 600;
      letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-tertiary);
      border-bottom: 1px solid var(--border-default); white-space: nowrap; background: rgba(255,255,255,0.02);
    }
    tbody tr {
      border-bottom: 1px solid rgba(255,255,255,0.04); cursor: pointer;
      transition: background 0.14s;
    }
    tbody tr:last-child { border-bottom: none; }
    tbody tr:hover { background: rgba(10,132,255,0.04); }
    tbody td { padding: 12px 14px; color: var(--text-secondary); font-feature-settings: "tnum"; }
    tbody td.td-id { color: var(--text-primary); font-weight: 500; font-family: var(--font-mono); font-size: 12px; }
    @keyframes rowFlash {
      0% { background: rgba(10,132,255,0.14); }
      100% { background: transparent; }
    }
    .row-flash { animation: rowFlash 1.1s var(--ease-out); }

    /* Status badge */
    .badge {
      display: inline-flex; align-items: center; gap: 5px; padding: 3px 9px;
      border-radius: var(--r-full); font-size: 11px; font-weight: 600; letter-spacing: 0.03em; white-space: nowrap;
    }
    .badge-dot { width: 5px; height: 5px; border-radius: 50%; flex-shrink: 0; }
    .badge-paid { background: rgba(48,209,88,0.12); border: 1px solid rgba(48,209,88,0.35); color: var(--green); box-shadow: 0 0 8px rgba(48,209,88,0.1); }
    .badge-paid .badge-dot { background: var(--green); }
    .badge-pending { background: rgba(255,159,10,0.12); border: 1px solid rgba(255,159,10,0.35); color: var(--orange); }
    .badge-pending .badge-dot { background: var(--orange); }
    .badge-cancelled { background: rgba(255,69,58,0.09); border: 1px solid rgba(255,69,58,0.28); color: var(--red); }
    .badge-cancelled .badge-dot { background: var(--red); }

    /* Skeleton */
    @keyframes shimmer { from { background-position: -400px 0; } to { background-position: 400px 0; } }
    .skeleton {
      background: linear-gradient(90deg, rgba(255,255,255,0.04) 25%, rgba(255,255,255,0.09) 50%, rgba(255,255,255,0.04) 75%);
      background-size: 400px 100%; animation: shimmer 1.4s ease-in-out infinite;
      border-radius: 4px; height: 14px;
    }

    /* Table footer */
    .table-footer {
      display: flex; align-items: center; justify-content: space-between;
      padding: 12px 16px; border-top: 1px solid var(--border-default);
    }
    .table-count { font-size: 12px; color: var(--text-tertiary); }
    .load-more-btn {
      padding: 7px 16px; font-size: 13px; font-weight: 500;
      background: rgba(10,132,255,0.12); border: 1px solid rgba(10,132,255,0.3);
      border-radius: var(--r-md); color: var(--blue); cursor: pointer; transition: all 0.15s;
    }
    .load-more-btn:hover { background: rgba(10,132,255,0.2); }
    .load-more-btn:disabled { opacity: 0.4; cursor: not-allowed; }

    /* \u2500\u2500 TESTING PANEL \u2500\u2500 */
    .test-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(380px, 1fr)); gap: 16px; }
    .test-card {
      background: var(--bg-card); border: 1px solid var(--border-default);
      border-top-color: var(--border-bright); border-radius: var(--r-xl); overflow: hidden;
    }
    .test-card-header {
      display: flex; align-items: center; gap: 10px;
      padding: 14px 16px; border-bottom: 1px solid var(--border-default);
    }
    .method-badge {
      padding: 2px 7px; border-radius: var(--r-sm); font-size: 10px; font-weight: 700;
      font-family: var(--font-mono); letter-spacing: 0.05em; flex-shrink: 0;
    }
    .method-get { background: rgba(48,209,88,0.15); color: var(--green); }
    .method-post { background: rgba(10,132,255,0.15); color: var(--blue); }
    .method-ws { background: rgba(191,90,242,0.15); color: var(--purple); }
    .method-patch { background: rgba(255,159,10,0.15); color: var(--orange); }
    .test-endpoint { font-size: 12px; font-family: var(--font-mono); color: var(--text-secondary); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .test-title { font-size: 13px; font-weight: 600; color: var(--text-primary); }
    .test-card-body { padding: 14px 16px; }
    .test-field { margin-bottom: 10px; }
    .test-field-label { font-size: 10px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-tertiary); margin-bottom: 5px; }
    .test-input, .test-textarea {
      width: 100%; padding: 8px 10px; font-size: 12px; font-family: var(--font-mono);
      background: var(--bg-input); border: 1px solid var(--border-default); border-radius: var(--r-sm);
      color: var(--text-primary); outline: none; transition: border-color 0.2s, background 0.2s;
    }
    .test-input:focus, .test-textarea:focus { background: var(--bg-input-focus); border-color: var(--border-focus); }
    .test-textarea { resize: vertical; min-height: 60px; max-height: 200px; }
    .test-actions { display: flex; gap: 8px; margin-top: 10px; align-items: center; }
    .test-run-btn {
      padding: 7px 16px; font-size: 12px; font-weight: 600; border-radius: var(--r-sm);
      background: var(--blue); color: #fff; border: none; cursor: pointer; transition: opacity 0.15s; flex-shrink: 0;
    }
    .test-run-btn:hover { opacity: 0.85; }
    .test-run-btn.running { opacity: 0.6; cursor: not-allowed; }
    .ws-btn {
      padding: 7px 14px; font-size: 12px; font-weight: 600; border-radius: var(--r-sm);
      border: 1px solid var(--border-default); color: var(--text-secondary); background: transparent;
      cursor: pointer; transition: all 0.15s; flex-shrink: 0;
    }
    .ws-btn.connected { color: var(--red); border-color: rgba(255,69,58,0.4); background: rgba(255,69,58,0.08); }
    .ws-btn:not(.connected):hover { color: var(--purple); border-color: rgba(191,90,242,0.4); }
    .autofill-hint { font-size: 11px; color: var(--text-tertiary); padding: 6px 10px; background: rgba(10,132,255,0.07); border: 1px solid rgba(10,132,255,0.2); border-radius: var(--r-sm); display: none; }
    .autofill-hint.visible { display: block; }
    .test-response {
      margin-top: 10px; background: rgba(0,0,0,0.35); border: 1px solid var(--border-default);
      border-radius: var(--r-sm); overflow: hidden; display: none;
    }
    .test-response.visible { display: block; }
    .test-response-head {
      display: flex; align-items: center; gap: 8px; padding: 7px 10px;
      border-bottom: 1px solid var(--border-default); background: rgba(255,255,255,0.02);
    }
    .status-pill { padding: 2px 7px; border-radius: var(--r-full); font-size: 10px; font-weight: 700; font-family: var(--font-mono); }
    .status-ok { background: rgba(48,209,88,0.15); color: var(--green); }
    .status-err { background: rgba(255,69,58,0.12); color: var(--red); }
    .timing { font-size: 10px; color: var(--text-tertiary); font-family: var(--font-mono); }
    .test-response-body {
      padding: 10px; font-family: var(--font-mono); font-size: 11px; color: var(--text-secondary);
      max-height: 180px; overflow-y: auto; white-space: pre; line-height: 1.55;
    }
    .json-key { color: var(--blue); }
    .json-str { color: var(--green); }
    .json-num { color: var(--orange); }
    .json-bool { color: var(--purple); }
    .json-null { color: var(--red); }

    /* \u2500\u2500 CONSOLE \u2500\u2500 */
    .console-toolbar { display: flex; gap: 8px; align-items: center; margin-bottom: 12px; }
    .console-level-btn {
      padding: 4px 10px; font-size: 11px; font-weight: 500; border-radius: var(--r-sm);
      border: 1px solid var(--border-default); background: transparent; color: var(--text-tertiary); cursor: pointer; transition: all 0.15s;
    }
    .console-level-btn.active { background: rgba(255,255,255,0.08); color: var(--text-primary); border-color: var(--border-bright); }
    .console-clear { margin-left: auto; padding: 4px 10px; font-size: 11px; font-weight: 500; border-radius: var(--r-sm); border: 1px solid var(--border-default); background: transparent; color: var(--text-tertiary); cursor: pointer; transition: all 0.15s; }
    .console-clear:hover { color: var(--red); border-color: rgba(255,69,58,0.4); }
    .console-box {
      background: rgba(0,0,0,0.5); border: 1px solid var(--border-default); border-radius: var(--r-xl);
      padding: 14px; font-family: var(--font-mono); font-size: 12px; height: calc(100vh - 220px);
      overflow-y: auto; line-height: 1.65;
    }
    .log-entry { display: flex; gap: 10px; padding: 3px 0; border-bottom: 1px solid rgba(255,255,255,0.03); }
    .log-entry:last-child { border-bottom: none; }
    .log-time { color: var(--text-tertiary); flex-shrink: 0; }
    .log-level { flex-shrink: 0; font-weight: 700; width: 42px; }
    .log-level-info { color: var(--blue); }
    .log-level-ok { color: var(--green); }
    .log-level-warn { color: var(--orange); }
    .log-level-error { color: var(--red); }
    .log-msg { color: var(--text-secondary); word-break: break-all; }
    .log-data { color: var(--text-tertiary); margin-top: 2px; white-space: pre; }

    /* \u2500\u2500 MODAL \u2500\u2500 */
    #modal-overlay {
      position: fixed; inset: 0; z-index: 200;
      background: rgba(0,0,0,0.65); backdrop-filter: blur(8px);
      display: flex; align-items: center; justify-content: center; padding: 20px;
    }
    .modal {
      background: var(--bg-modal); border: 1px solid var(--border-bright);
      border-radius: var(--r-2xl); width: 100%; max-width: 520px;
      box-shadow: var(--shadow-xl); overflow: hidden;
      animation: modalIn 0.28s var(--ease-out) forwards;
    }
    @keyframes modalIn {
      from { opacity: 0; transform: scale(0.94) translateY(12px); }
      to { opacity: 1; transform: scale(1) translateY(0); }
    }
    .modal-header {
      padding: 20px 24px 16px; border-bottom: 1px solid var(--border-default);
      display: flex; align-items: flex-start; gap: 12px;
    }
    .modal-tid { font-size: 13px; font-family: var(--font-mono); color: var(--blue); font-weight: 600; }
    .modal-title { font-size: 18px; font-weight: 700; margin-top: 2px; }
    .modal-close { margin-left: auto; width: 28px; height: 28px; border-radius: 50%; background: rgba(255,255,255,0.08); border: none; cursor: pointer; color: var(--text-secondary); font-size: 16px; display: flex; align-items: center; justify-content: center; transition: all 0.15s; flex-shrink: 0; }
    .modal-close:hover { background: rgba(255,255,255,0.15); color: var(--text-primary); }
    .modal-body { padding: 20px 24px; max-height: 55vh; overflow-y: auto; }
    .modal-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
    .modal-grid .full { grid-column: 1 / -1; }
    .modal-field { }
    .modal-label { font-size: 10px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-tertiary); margin-bottom: 6px; }
    .modal-input, .modal-select {
      width: 100%; padding: 9px 12px; font-size: 13px; font-family: var(--font-ui);
      background: var(--bg-input); border: 1px solid var(--border-default); border-radius: var(--r-sm);
      color: var(--text-primary); outline: none; transition: all 0.2s;
    }
    .modal-input:focus, .modal-select:focus { background: var(--bg-input-focus); border-color: var(--border-focus); }
    .modal-select option { background: #1c1c1e; }
    .modal-footer {
      padding: 16px 24px; border-top: 1px solid var(--border-default);
      display: flex; align-items: center; gap: 8px;
    }
    .modal-action-btn {
      padding: 8px 16px; font-size: 13px; font-weight: 600; border-radius: var(--r-sm);
      cursor: pointer; transition: all 0.15s; border: none;
    }
    .btn-paid { background: rgba(48,209,88,0.15); color: var(--green); border: 1px solid rgba(48,209,88,0.3); }
    .btn-paid:hover { background: rgba(48,209,88,0.25); }
    .btn-cancel { background: rgba(255,69,58,0.1); color: var(--red); border: 1px solid rgba(255,69,58,0.25); }
    .btn-cancel:hover { background: rgba(255,69,58,0.18); }
    .modal-footer-spacer { flex: 1; }
    .btn-discard { padding: 8px 14px; font-size: 13px; font-weight: 500; border-radius: var(--r-sm); cursor: pointer; background: transparent; border: 1px solid var(--border-default); color: var(--text-secondary); transition: all 0.15s; }
    .btn-discard:hover { border-color: var(--border-bright); color: var(--text-primary); }
    .btn-save { padding: 8px 18px; font-size: 13px; font-weight: 600; border-radius: var(--r-sm); cursor: pointer; background: var(--blue); border: none; color: #fff; transition: opacity 0.15s; }
    .btn-save:hover { opacity: 0.85; }
    .btn-save:disabled { opacity: 0.4; cursor: not-allowed; }

    /* \u2500\u2500 TOASTS \u2500\u2500 */
    #toast-stack { position: fixed; bottom: 24px; right: 24px; display: flex; flex-direction: column; gap: 8px; z-index: 500; pointer-events: none; }
    .toast {
      padding: 12px 16px; border-radius: var(--r-md); font-size: 13px; font-weight: 500;
      box-shadow: var(--shadow-lg); backdrop-filter: blur(20px);
      animation: toastIn 0.25s var(--ease-spring) forwards;
      border: 1px solid rgba(255,255,255,0.1);
    }
    .toast-success { background: rgba(30,62,28,0.94); color: var(--green); }
    .toast-error { background: rgba(62,20,18,0.94); color: var(--red); }
    .toast-info { background: rgba(12,30,60,0.94); color: var(--blue); }
    @keyframes toastIn { from { opacity: 0; transform: translateY(12px) scale(0.96); } to { opacity: 1; transform: translateY(0) scale(1); } }
    @keyframes toastOut { to { opacity: 0; transform: translateY(8px) scale(0.95); } }

    @media (max-width: 768px) {
      .stats-grid { grid-template-columns: repeat(2, 1fr); }
      .test-grid { grid-template-columns: 1fr; }
      .modal-grid { grid-template-columns: 1fr; }
      .modal-grid .full { grid-column: 1; }
      header { padding: 0 16px; }
      main { padding: 16px; }
    }
  </style>
</head>
<body>

<!-- LOGIN -->
<div id="login-screen">
  <div class="login-card" id="login-card">
    <div class="login-logo">&#9632; PayAdmin</div>
    <div class="login-sub">Sign in to access the dashboard</div>
    <div class="login-label">Password</div>
    <input class="login-input" id="login-pw" type="password" placeholder="Enter password" autocomplete="current-password">
    <button class="login-btn" id="login-btn">Sign In</button>
    <div class="login-err" id="login-err"></div>
  </div>
</div>

<!-- DASHBOARD -->
<div id="dashboard" style="display:none">
  <header>
    <div class="header-logo">&#9632; PayAdmin</div>
    <nav class="nav-tabs">
      <button class="nav-tab active" data-tab="overview">Overview</button>
      <button class="nav-tab" data-tab="testing">Testing</button>
      <button class="nav-tab" data-tab="console">Console</button>
    </nav>
    <div class="header-spacer"></div>
    <span class="rt-label">Live</span>
    <div class="rt-dot disconnected" id="rt-dot"></div>
    <button class="header-btn" id="refresh-btn">&#8635; Refresh</button>
    <button class="header-btn" id="signout-btn">Sign Out</button>
  </header>

  <main>
    <!-- OVERVIEW TAB -->
    <div id="tab-overview">
      <div class="stats-grid">
        <div class="stat-card active-filter" data-key="all" onclick="Dashboard.setFilter('')">
          <div class="stat-label">Total Tickets</div>
          <div class="stat-value" id="stat-total">&#8212;</div>
        </div>
        <div class="stat-card" data-key="paid" onclick="Dashboard.setFilter('paid')">
          <div class="stat-label">Paid</div>
          <div class="stat-value" id="stat-paid">&#8212;</div>
        </div>
        <div class="stat-card" data-key="pending" onclick="Dashboard.setFilter('pending')">
          <div class="stat-label">Pending</div>
          <div class="stat-value" id="stat-pending">&#8212;</div>
        </div>
        <div class="stat-card" data-key="cancelled" onclick="Dashboard.setFilter('cancelled')">
          <div class="stat-label">Cancelled</div>
          <div class="stat-value" id="stat-cancelled">&#8212;</div>
        </div>
      </div>

      <div class="toolbar">
        <div class="search-wrap">
          <svg class="search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
          <input class="search-input" id="search-input" type="text" placeholder="Search tickets, names, UPI IDs...">
        </div>
        <div class="filter-pills">
          <button class="pill active" data-status="" onclick="Dashboard.setFilter('')">All</button>
          <button class="pill" data-status="paid" onclick="Dashboard.setFilter('paid')">Paid</button>
          <button class="pill" data-status="pending" onclick="Dashboard.setFilter('pending')">Pending</button>
          <button class="pill" data-status="cancelled" onclick="Dashboard.setFilter('cancelled')">Cancelled</button>
        </div>
        <div class="date-range">
          <input class="date-input" id="date-from" type="date" title="From date">
          <span class="date-sep">&#8594;</span>
          <input class="date-input" id="date-to" type="date" title="To date">
          <button class="date-clear" id="date-clear" title="Clear dates">&#10005;</button>
        </div>
      </div>

      <div class="table-card">
        <div class="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Ticket ID</th>
                <th>Amount</th>
                <th>Status</th>
                <th>Sender</th>
                <th>UPI ID</th>
                <th>RRN</th>
                <th>Created</th>
                <th>Paid At</th>
              </tr>
            </thead>
            <tbody id="ticket-tbody"></tbody>
          </table>
        </div>
        <div class="table-footer">
          <div class="table-count" id="table-count">Loading...</div>
          <button class="load-more-btn" id="load-more-btn" style="display:none" onclick="Dashboard.loadMore()">Load More</button>
        </div>
      </div>
    </div>

    <!-- TESTING TAB -->
    <div id="tab-testing" style="display:none">
      <div class="test-grid">

        <!-- 1. Create Ticket -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-post">POST</span>
            <span class="test-endpoint">/api/ticket</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">Create Ticket</div>
            <div class="autofill-hint" id="hint-create"></div>
            <div class="test-field">
              <div class="test-field-label">Amount (\u20B9)</div>
              <input class="test-input" id="tc-amount" type="number" value="500" min="1">
            </div>
            <div class="test-actions">
              <button class="test-run-btn" onclick="Testing.createTicket()">Run</button>
            </div>
            <div class="test-response" id="resp-create"></div>
          </div>
        </div>

        <!-- 2. Standard SMS Webhook -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-post">POST</span>
            <span class="test-endpoint">/api/webhook</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">SMS Webhook (Standard)</div>
            <div class="autofill-hint" id="hint-sms" class="autofill-hint"></div>
            <div class="test-field">
              <div class="test-field-label">Webhook Secret</div>
              <input class="test-input" id="tw-secret" type="text" value="REDACTED">
            </div>
            <div class="test-field">
              <div class="test-field-label">SMS Body</div>
              <textarea class="test-textarea" id="tw-sms" rows="3">TICKET1234567890
Test Sender paid you &#8377;500</textarea>
            </div>
            <div class="test-actions">
              <button class="test-run-btn" onclick="Testing.smsWebhook()">Run</button>
            </div>
            <div class="test-response" id="resp-sms"></div>
          </div>
        </div>

        <!-- 3. Kotak SMS Webhook -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-post">POST</span>
            <span class="test-endpoint">/api/webhook</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">SMS Webhook (Kotak)</div>
            <div class="autofill-hint" id="hint-kotak"></div>
            <div class="test-field">
              <div class="test-field-label">Webhook Secret</div>
              <input class="test-input" id="tk-secret" type="text" value="REDACTED">
            </div>
            <div class="test-field">
              <div class="test-field-label">Kotak SMS Body</div>
              <textarea class="test-textarea" id="tk-sms" rows="3">Confirmed payment for Received Rs.500.00 in your Kotak Bank AC X4959 from testuser@oksbi on 15-03-26.UPI Ref:123456789012. KOTAK</textarea>
            </div>
            <div class="test-actions">
              <button class="test-run-btn" onclick="Testing.kotakWebhook()">Run</button>
            </div>
            <div class="test-response" id="resp-kotak"></div>
          </div>
        </div>

        <!-- 4. Email Webhook -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-post">POST</span>
            <span class="test-endpoint">/api/email-webhook</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">Email Webhook (Slice)</div>
            <div class="autofill-hint" id="hint-email"></div>
            <div class="test-field">
              <div class="test-field-label">Email Secret</div>
              <input class="test-input" id="te-secret" type="text" value="your-email-secret">
            </div>
            <div class="test-field">
              <div class="test-field-label">From</div>
              <input class="test-input" id="te-from" value="no-reply@sliceit.com">
            </div>
            <div class="test-field">
              <div class="test-field-label">Subject</div>
              <input class="test-input" id="te-subject" value="&#8377;500 received via UPI">
            </div>
            <div class="test-field">
              <div class="test-field-label">Body (HTML)</div>
              <textarea class="test-textarea" id="te-body" rows="4">&lt;table&gt;&lt;tr&gt;&lt;td&gt;From&lt;/td&gt;&lt;td&gt;TEST SENDER&lt;/td&gt;&lt;/tr&gt;&lt;tr&gt;&lt;td&gt;RRN&lt;/td&gt;&lt;td&gt;123456789012&lt;/td&gt;&lt;/tr&gt;&lt;/table&gt;</textarea>
            </div>
            <div class="test-actions">
              <button class="test-run-btn" onclick="Testing.emailWebhook()">Run</button>
            </div>
            <div class="test-response" id="resp-email"></div>
          </div>
        </div>

        <!-- 5. Status Check -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-get">GET</span>
            <span class="test-endpoint">/api/status/:id</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">Check Ticket Status</div>
            <div class="autofill-hint" id="hint-status"></div>
            <div class="test-field">
              <div class="test-field-label">Ticket ID</div>
              <input class="test-input" id="ts-id" type="text" placeholder="TICKET1234567890">
            </div>
            <div class="test-actions">
              <button class="test-run-btn" onclick="Testing.checkStatus()">Run</button>
            </div>
            <div class="test-response" id="resp-status"></div>
          </div>
        </div>

        <!-- 6. Ping WebSocket -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-ws">WS</span>
            <span class="test-endpoint">/api/ping</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">Ping WebSocket</div>
            <div class="test-actions">
              <button class="ws-btn" id="ws-ping-btn" onclick="Testing.togglePingWs()">Connect</button>
            </div>
            <div class="test-response" id="resp-ping"></div>
          </div>
        </div>

        <!-- 7. Ticket WebSocket -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-ws">WS</span>
            <span class="test-endpoint">/api/ws</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">Ticket WebSocket</div>
            <div class="autofill-hint" id="hint-ticketws"></div>
            <div class="test-field">
              <div class="test-field-label">Ticket ID</div>
              <input class="test-input" id="ws-ticket-id" type="text" placeholder="TICKET1234567890">
            </div>
            <div class="test-actions">
              <button class="ws-btn" id="ws-ticket-btn" onclick="Testing.toggleTicketWs()">Connect</button>
            </div>
            <div class="test-response" id="resp-ticketws"></div>
          </div>
        </div>

      </div>
    </div>

    <!-- CONSOLE TAB -->
    <div id="tab-console" style="display:none">
      <div class="console-toolbar">
        <button class="console-level-btn active" data-level="" onclick="AppConsole.setFilter('')">All</button>
        <button class="console-level-btn" data-level="info" onclick="AppConsole.setFilter('info')">Info</button>
        <button class="console-level-btn" data-level="ok" onclick="AppConsole.setFilter('ok')">OK</button>
        <button class="console-level-btn" data-level="warn" onclick="AppConsole.setFilter('warn')">Warn</button>
        <button class="console-level-btn" data-level="error" onclick="AppConsole.setFilter('error')">Error</button>
        <button class="console-clear" onclick="AppConsole.clear()">Clear</button>
      </div>
      <div class="console-box" id="console-box"></div>
    </div>
  </main>
</div>

<!-- MODAL -->
<div id="modal-overlay" style="display:none">
  <div class="modal" id="modal">
    <div class="modal-header">
      <div>
        <div class="modal-tid" id="modal-tid"></div>
        <div class="modal-title" id="modal-title">Ticket Details</div>
      </div>
      <button class="modal-close" onclick="Modal.close()">&#10005;</button>
    </div>
    <div class="modal-body">
      <div class="modal-grid">
        <div class="modal-field full">
          <div class="modal-label">Status</div>
          <select class="modal-select" id="m-status">
            <option value="pending">Pending</option>
            <option value="paid">Paid</option>
            <option value="cancelled">Cancelled</option>
          </select>
        </div>
        <div class="modal-field">
          <div class="modal-label">Amount (\u20B9)</div>
          <input class="modal-input" id="m-amount" type="number" step="0.01">
        </div>
        <div class="modal-field">
          <div class="modal-label">Sender Name</div>
          <input class="modal-input" id="m-sender" type="text">
        </div>
        <div class="modal-field">
          <div class="modal-label">RRN</div>
          <input class="modal-input" id="m-rrn" type="text">
        </div>
        <div class="modal-field">
          <div class="modal-label">UPI ID</div>
          <input class="modal-input" id="m-upi" type="text">
        </div>
        <div class="modal-field full">
          <div class="modal-label">Paid At (ISO)</div>
          <input class="modal-input" id="m-paidat" type="text" placeholder="e.g. 2024-03-15T10:30:00.000Z">
        </div>
        <div class="modal-field full">
          <div class="modal-label">Created At</div>
          <input class="modal-input" id="m-createdat" type="text" readonly style="opacity:0.5;cursor:not-allowed">
        </div>
      </div>
    </div>
    <div class="modal-footer">
      <button class="modal-action-btn btn-paid" onclick="Modal.markPaid()">Mark as Paid</button>
      <button class="modal-action-btn btn-cancel" onclick="Modal.cancelTicket()">Cancel Ticket</button>
      <div class="modal-footer-spacer"></div>
      <button class="btn-discard" onclick="Modal.discard()">Discard</button>
      <button class="btn-save" id="modal-save-btn" onclick="Modal.save()">Save</button>
    </div>
  </div>
</div>

<div id="toast-stack"></div>

<script>
(function() {

// \u2500\u2500\u2500 STATE \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
var State = {
  pw: '',
  allLoaded: [],
  stats: { total: 0, paid: 0, pending: 0, cancelled: 0 },
  hasMore: false,
  currentPage: 0,
  PAGE_SIZE: 10,
  statusFilter: '',
  search: '',
  searchTimer: null,
  dateFrom: '',
  dateTo: '',
  lastCreated: null,  // { ticketId, amount }
  logs: [],
  logFilter: '',
  activeTab: 'overview',
  pingWs: null,
  ticketWs: null,
  modalTicketId: null
};

// \u2500\u2500\u2500 UTIL \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
var Util = {
  escape: function(s) {
    return String(s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  },
  fmtAmount: function(n) {
    return '\\u20B9' + Number(n).toFixed(2);
  },
  relativeTime: function(iso) {
    if (!iso) return '\u2014';
    var diff = Date.now() - new Date(iso).getTime();
    var s = Math.floor(diff / 1000);
    if (s < 60) return s + 's ago';
    var m = Math.floor(s / 60);
    if (m < 60) return m + 'm ago';
    var h = Math.floor(m / 60);
    if (h < 24) return h + 'h ago';
    return Math.floor(h / 24) + 'd ago';
  },
  fmtDateShort: function(iso) {
    if (!iso) return '\u2014';
    var d = new Date(iso);
    return d.toLocaleDateString('en-IN', { day:'2-digit', month:'short', year:'2-digit' })
      + ' ' + d.toLocaleTimeString('en-IN', { hour:'2-digit', minute:'2-digit', hour12: true });
  },
  colorizeJson: function(obj) {
    var json = typeof obj === 'string' ? obj : JSON.stringify(obj, null, 2);
    return json.split('
').map(function(line) {
      // Key: leading "word":
      line = line.replace(/^(s*)("(?:[^"]*)")(s*:)/, function(m, sp, k, c) {
        return sp + '<span class="json-key">' + k + '</span>' + c;
      });
      // String value after colon
      line = line.replace(/(:s*)("(?:[^"]*)")/, function(m, c, v) {
        return c + '<span class="json-str">' + v + '</span>';
      });
      // Number value
      line = line.replace(/(:s*)([-]?[0-9]+[.]?[0-9]*)/, function(m, c, v) {
        return c + '<span class="json-num">' + v + '</span>';
      });
      // Boolean
      line = line.replace(/(:s*)(true|false)/, function(m, c, v) {
        return c + '<span class="json-bool">' + v + '</span>';
      });
      // Null
      line = line.replace(/(:s*)(null)/, function(m, c, v) {
        return c + '<span class="json-null">' + v + '</span>';
      });
      return line;
    }).join('
');
  }
};

// \u2500\u2500\u2500 API \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
var API = {
  buildTicketsUrl: function() {
    var p = new URLSearchParams();
    p.set('page', State.currentPage.toString());
    p.set('pageSize', State.PAGE_SIZE.toString());
    if (State.search.trim()) p.set('search', State.search.trim());
    if (State.statusFilter) p.set('status', State.statusFilter);
    if (State.dateFrom) p.set('dateFrom', State.dateFrom);
    if (State.dateTo) p.set('dateTo', State.dateTo);
    return '/api/admin/tickets?' + p.toString();
  },
  headers: function() {
    return { 'Content-Type': 'application/json', 'X-Admin-Password': State.pw };
  },
  get: async function(url) {
    var t0 = Date.now();
    AppConsole.log('info', 'GET ' + url);
    try {
      var r = await fetch(url, { headers: this.headers() });
      var data = await r.json();
      AppConsole.log(r.ok ? 'ok' : 'warn', 'GET ' + url + ' -> ' + r.status + ' (' + (Date.now()-t0) + 'ms)');
      return { ok: r.ok, status: r.status, data: data, ms: Date.now()-t0 };
    } catch(e) {
      AppConsole.log('error', 'GET ' + url + ' failed: ' + e.message);
      return { ok: false, status: 0, data: { error: e.message }, ms: Date.now()-t0 };
    }
  },
  post: async function(url, body, extraHeaders) {
    var t0 = Date.now();
    AppConsole.log('info', 'POST ' + url);
    try {
      var hdrs = Object.assign({}, this.headers(), extraHeaders || {});
      var r = await fetch(url, { method: 'POST', headers: hdrs, body: JSON.stringify(body) });
      var data = await r.json();
      AppConsole.log(r.ok ? 'ok' : 'warn', 'POST ' + url + ' -> ' + r.status + ' (' + (Date.now()-t0) + 'ms)');
      return { ok: r.ok, status: r.status, data: data, ms: Date.now()-t0 };
    } catch(e) {
      AppConsole.log('error', 'POST ' + url + ' failed: ' + e.message);
      return { ok: false, status: 0, data: { error: e.message }, ms: Date.now()-t0 };
    }
  },
  patch: async function(url, body) {
    var t0 = Date.now();
    AppConsole.log('info', 'PATCH ' + url);
    try {
      var r = await fetch(url, { method: 'PATCH', headers: this.headers(), body: JSON.stringify(body) });
      var data = await r.json();
      AppConsole.log(r.ok ? 'ok' : 'warn', 'PATCH ' + url + ' -> ' + r.status + ' (' + (Date.now()-t0) + 'ms)');
      return { ok: r.ok, status: r.status, data: data, ms: Date.now()-t0 };
    } catch(e) {
      AppConsole.log('error', 'PATCH ' + url + ' failed: ' + e.message);
      return { ok: false, status: 0, data: { error: e.message }, ms: Date.now()-t0 };
    }
  }
};

// \u2500\u2500\u2500 AUTH \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
var Auth = {
  init: function() {
    var saved = sessionStorage.getItem('adminPw');
    if (saved) { State.pw = saved; this.enter(); return; }
    document.getElementById('login-pw').addEventListener('keydown', function(e) {
      if (e.key === 'Enter') Auth.login();
    });
    document.getElementById('login-btn').addEventListener('click', Auth.login.bind(Auth));
  },
  login: async function() {
    var pw = document.getElementById('login-pw').value;
    if (!pw) return;
    State.pw = pw;
    var res = await API.get('/api/admin/tickets?page=0&pageSize=1');
    if (!res.ok && res.status === 401) {
      State.pw = '';
      document.getElementById('login-err').textContent = 'Incorrect password.';
      var card = document.getElementById('login-card');
      card.classList.remove('shake');
      void card.offsetWidth;
      card.classList.add('shake');
      return;
    }
    sessionStorage.setItem('adminPw', pw);
    this.enter();
  },
  enter: function() {
    document.getElementById('login-screen').style.display = 'none';
    document.getElementById('dashboard').style.display = 'flex';
    Dashboard.loadTickets(true, true);
    Realtime.connect();
  },
  logout: function() {
    sessionStorage.removeItem('adminPw');
    State.pw = '';
    Realtime.disconnect();
    document.getElementById('dashboard').style.display = 'none';
    document.getElementById('login-screen').style.display = 'flex';
    document.getElementById('login-pw').value = '';
    document.getElementById('login-err').textContent = '';
  }
};

// \u2500\u2500\u2500 REALTIME \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
var Realtime = {
  ws: null,
  reconnTimer: null,
  forceClosed: false,
  connect: function() {
    this.forceClosed = false;
    clearTimeout(this.reconnTimer);
    try {
      var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
      var url = proto + '//' + location.host + '/api/admin/ws?pw=' + encodeURIComponent(State.pw);
      this.ws = new WebSocket(url);
      this.ws.onopen = function() {
        Realtime.setStatus(true);
        AppConsole.log('ok', '[Realtime] Connected to live feed');
      };
      this.ws.onmessage = function(e) {
        try {
          var msg = JSON.parse(e.data);
          if (msg.type === 'ticket_update') Realtime.handleUpdate(msg.ticket, msg.action);
        } catch(_) {}
      };
      this.ws.onclose = function() {
        Realtime.setStatus(false);
        if (!Realtime.forceClosed) {
          AppConsole.log('warn', '[Realtime] Disconnected \u2014 reconnecting in 5s');
          Realtime.reconnTimer = setTimeout(function() { Realtime.connect(); }, 5000);
        }
      };
      this.ws.onerror = function() {
        Realtime.setStatus(false);
        AppConsole.log('error', '[Realtime] WebSocket error');
      };
    } catch(e) {
      AppConsole.log('error', '[Realtime] Could not connect: ' + e.message);
    }
  },
  disconnect: function() {
    this.forceClosed = true;
    clearTimeout(this.reconnTimer);
    try { if (this.ws) this.ws.close(); } catch(_) {}
    this.ws = null;
    this.setStatus(false);
  },
  handleUpdate: function(ticket, action) {
    AppConsole.log('info', '[Realtime] ' + action + ': ' + ticket.ticketId + ' (' + ticket.status + ')');
    if (action === 'delete') {
      State.allLoaded = State.allLoaded.filter(function(t) { return t.id !== ticket.id; });
      if (State.stats.total > 0) State.stats.total--;
    } else if (action === 'create') {
      State.allLoaded.unshift(ticket);
      State.stats.total = (State.stats.total || 0) + 1;
      State.stats[ticket.status] = (State.stats[ticket.status] || 0) + 1;
      UI.toast('New ticket: ' + ticket.ticketId, 'info');
    } else {
      var idx = -1;
      for (var i = 0; i < State.allLoaded.length; i++) {
        if (State.allLoaded[i].id === ticket.id) { idx = i; break; }
      }
      if (idx >= 0) {
        var old = State.allLoaded[idx];
        if (old.status !== ticket.status) {
          State.stats[old.status] = Math.max(0, (State.stats[old.status] || 0) - 1);
          State.stats[ticket.status] = (State.stats[ticket.status] || 0) + 1;
          if (ticket.status === 'paid') UI.toast('Payment received: ' + ticket.ticketId, 'success');
        }
        State.allLoaded[idx] = ticket;
      } else {
        // Not in current page \u2014 just update stats footer silently
      }
    }
    Dashboard.renderStats();
    Dashboard.renderTickets();
    // Flash updated row
    setTimeout(function() {
      var row = document.getElementById('row-' + ticket.id);
      if (row) {
        row.classList.remove('row-flash');
        void row.offsetWidth;
        row.classList.add('row-flash');
      }
    }, 30);
  },
  setStatus: function(connected) {
    var dot = document.getElementById('rt-dot');
    if (dot) dot.className = 'rt-dot ' + (connected ? 'connected' : 'disconnected');
  }
};

// \u2500\u2500\u2500 DASHBOARD \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
var Dashboard = {
  loading: false,
  prevStats: { total: 0, paid: 0, pending: 0, cancelled: 0 },

  setFilter: function(status) {
    State.statusFilter = status;
    // Update pills
    document.querySelectorAll('.pill').forEach(function(el) {
      el.classList.toggle('active', el.dataset.status === status);
    });
    // Update stat cards
    document.querySelectorAll('.stat-card').forEach(function(el) {
      var key = el.dataset.key;
      var matches = (status === '' && key === 'all') || (status === key);
      el.classList.toggle('active-filter', matches);
    });
    this.loadTickets(true, true);
  },

  loadTickets: async function(reset, spin) {
    if (this.loading) return;
    this.loading = true;
    if (reset) { State.currentPage = 0; State.allLoaded = []; }
    if (spin) UI.showTableLoading();
    var res = await API.get(API.buildTicketsUrl());
    this.loading = false;
    if (!res.ok) { UI.toast('Failed to load tickets', 'error'); return; }
    var d = res.data;
    State.stats = d.stats || State.stats;
    State.hasMore = d.hasMore || false;
    if (reset) {
      State.allLoaded = d.tickets || [];
    } else {
      State.allLoaded = State.allLoaded.concat(d.tickets || []);
    }
    this.renderStats();
    this.renderTickets();
  },

  loadMore: async function() {
    var btn = document.getElementById('load-more-btn');
    if (btn) { btn.disabled = true; btn.textContent = 'Loading...'; }
    State.currentPage++;
    await this.loadTickets(false, false);
    if (btn) { btn.disabled = false; btn.textContent = 'Load More'; }
  },

  renderStats: function() {
    var s = State.stats;
    var keys = ['total','paid','pending','cancelled'];
    for (var i = 0; i < keys.length; i++) {
      var k = keys[i];
      var el = document.getElementById('stat-' + k);
      if (el) {
        var prev = this.prevStats[k] || 0;
        var next = s[k] || 0;
        if (prev !== next) UI.animateCounter(el, prev, next);
        else el.textContent = next;
        this.prevStats[k] = next;
      }
    }
  },

  renderTickets: function() {
    var tbody = document.getElementById('ticket-tbody');
    if (!tbody) return;
    var items = State.allLoaded;
    if (items.length === 0) {
      tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;padding:40px;color:var(--text-tertiary)">No tickets found</td></tr>';
      document.getElementById('table-count').textContent = '0 tickets';
      document.getElementById('load-more-btn').style.display = 'none';
      return;
    }
    var html = '';
    for (var i = 0; i < items.length; i++) {
      var t = items[i];
      var badgeCls = 'badge badge-' + t.status;
      var dot = '<span class="badge-dot"></span>';
      html += '<tr id="row-' + Util.escape(t.id) + '" data-id="' + Util.escape(t.id) + '" onclick="Modal.open(this.dataset.id)">';
      html += '<td class="td-id">' + Util.escape(t.ticketId) + '</td>';
      html += '<td>' + Util.fmtAmount(t.amount) + '</td>';
      html += '<td><span class="' + badgeCls + '">' + dot + Util.escape(t.status) + '</span></td>';
      html += '<td>' + Util.escape(t.senderName || '\u2014') + '</td>';
      html += '<td style="font-family:var(--font-mono);font-size:11px">' + Util.escape(t.upiId || '\u2014') + '</td>';
      html += '<td style="font-family:var(--font-mono);font-size:11px">' + Util.escape(t.rrn || '\u2014') + '</td>';
      html += '<td title="' + Util.escape(t.createdAt) + '">' + Util.relativeTime(t.createdAt) + '</td>';
      html += '<td title="' + Util.escape(t.paidAt || '') + '">' + (t.paidAt ? Util.fmtDateShort(t.paidAt) : '\u2014') + '</td>';
      html += '</tr>';
    }
    tbody.innerHTML = html;

    var total = State.stats.total || 0;
    var showing = items.length;
    document.getElementById('table-count').textContent = 'Showing ' + showing + ' of ' + total + ' tickets';

    var loadBtn = document.getElementById('load-more-btn');
    loadBtn.style.display = State.hasMore ? 'block' : 'none';
  }
};

// \u2500\u2500\u2500 MODAL \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
var Modal = {
  original: null,
  open: function(id) {
    var ticket = null;
    for (var i = 0; i < State.allLoaded.length; i++) {
      if (State.allLoaded[i].id === id) { ticket = State.allLoaded[i]; break; }
    }
    if (!ticket) return;
    State.modalTicketId = id;
    this.original = JSON.parse(JSON.stringify(ticket));
    document.getElementById('modal-tid').textContent = ticket.ticketId;
    document.getElementById('modal-title').textContent = 'Ticket Details';
    document.getElementById('m-status').value = ticket.status;
    document.getElementById('m-amount').value = ticket.amount;
    document.getElementById('m-sender').value = ticket.senderName || '';
    document.getElementById('m-rrn').value = ticket.rrn || '';
    document.getElementById('m-upi').value = ticket.upiId || '';
    document.getElementById('m-paidat').value = ticket.paidAt || '';
    document.getElementById('m-createdat').value = ticket.createdAt || '';
    document.getElementById('modal-overlay').style.display = 'flex';
  },
  close: function() {
    document.getElementById('modal-overlay').style.display = 'none';
    State.modalTicketId = null;
    this.original = null;
  },
  discard: function() { this.close(); },
  save: async function() {
    var id = State.modalTicketId;
    if (!id) return;
    var payload = {
      status: document.getElementById('m-status').value,
      amount: parseFloat(document.getElementById('m-amount').value),
      senderName: document.getElementById('m-sender').value || undefined,
      rrn: document.getElementById('m-rrn').value || undefined,
      upiId: document.getElementById('m-upi').value || undefined,
      paidAt: document.getElementById('m-paidat').value || undefined
    };
    var btn = document.getElementById('modal-save-btn');
    btn.disabled = true; btn.textContent = 'Saving...';
    var res = await API.patch('/api/admin/tickets/' + encodeURIComponent(id), payload);
    btn.disabled = false; btn.textContent = 'Save';
    if (res.ok) {
      UI.toast('Ticket updated', 'success');
      this.close();
      Dashboard.loadTickets(true, false);
    } else {
      UI.toast('Update failed: ' + (res.data.error || res.status), 'error');
    }
  },
  markPaid: async function() {
    var id = State.modalTicketId;
    if (!id) return;
    var res = await API.post('/api/admin/tickets/' + encodeURIComponent(id) + '/mark-paid', {});
    if (res.ok) { UI.toast('Marked as paid', 'success'); this.close(); Dashboard.loadTickets(true, false); }
    else UI.toast('Failed: ' + (res.data.error || res.status), 'error');
  },
  cancelTicket: async function() {
    var id = State.modalTicketId;
    if (!id) return;
    var res = await API.post('/api/admin/tickets/' + encodeURIComponent(id) + '/cancel', {});
    if (res.ok) { UI.toast('Ticket cancelled', 'success'); this.close(); Dashboard.loadTickets(true, false); }
    else UI.toast('Failed: ' + (res.data.error || res.status), 'error');
  }
};

// \u2500\u2500\u2500 TESTING \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
var Testing = {
  showResponse: function(elId, res) {
    var el = document.getElementById(elId);
    if (!el) return;
    var pillCls = res.ok ? 'status-pill status-ok' : 'status-pill status-err';
    var html = '<div class="test-response-head">'
      + '<span class="' + pillCls + '">' + (res.status || 'ERR') + '</span>'
      + '<span class="timing">' + res.ms + 'ms</span>'
      + '</div>'
      + '<div class="test-response-body">' + Util.colorizeJson(res.data) + '</div>';
    el.innerHTML = html;
    el.classList.add('visible');
  },

  autoFill: function(ticketId, amount) {
    State.lastCreated = { ticketId: ticketId, amount: amount };
    var amtStr = amount.toFixed(2);
    var intAmt = Math.floor(amount);
    var decAmt = Math.round((amount - intAmt) * 100);
    var decStr = String(decAmt).padStart(2, '0');
    var today = new Date();
    var dd = String(today.getDate()).padStart(2,'0');
    var mm = String(today.getMonth()+1).padStart(2,'0');
    var yy = String(today.getFullYear()).slice(-2);
    var dateStr = dd + '-' + mm + '-' + yy;

    // SMS
    var smsEl = document.getElementById('tw-sms');
    if (smsEl) smsEl.value = ticketId + '\\nTest Sender paid you \\u20B9' + amtStr;
    var hintSms = document.getElementById('hint-sms');
    if (hintSms) { hintSms.textContent = '\\u2713 Auto-filled with ticket ' + ticketId; hintSms.classList.add('visible'); }

    // Kotak SMS
    var kotakEl = document.getElementById('tk-sms');
    if (kotakEl) kotakEl.value = 'Confirmed payment for Received Rs.' + amtStr + ' in your Kotak Bank AC X4959 from testuser@oksbi on ' + dateStr + '.UPI Ref:' + (123456789000 + decAmt) + '. KOTAK';
    var hintKotak = document.getElementById('hint-kotak');
    if (hintKotak) { hintKotak.textContent = '\\u2713 Auto-filled with amount ' + amtStr; hintKotak.classList.add('visible'); }

    // Email
    var subEl = document.getElementById('te-subject');
    var bodyEl = document.getElementById('te-body');
    if (subEl) subEl.value = '\\u20B9' + amtStr + ' received via UPI';
    if (bodyEl) bodyEl.value = '<table><tr><td>From</td><td>TEST SENDER</td></tr><tr><td>RRN</td><td>123456789' + decStr + '</td></tr></table>';
    var hintEmail = document.getElementById('hint-email');
    if (hintEmail) { hintEmail.textContent = '\\u2713 Auto-filled with amount ' + amtStr; hintEmail.classList.add('visible'); }

    // Status check
    var statusEl = document.getElementById('ts-id');
    if (statusEl) statusEl.value = ticketId;
    var hintStatus = document.getElementById('hint-status');
    if (hintStatus) { hintStatus.textContent = '\\u2713 Auto-filled with ' + ticketId; hintStatus.classList.add('visible'); }

    // Ticket WS
    var wsEl = document.getElementById('ws-ticket-id');
    if (wsEl) wsEl.value = ticketId;
    var hintWs = document.getElementById('hint-ticketws');
    if (hintWs) { hintWs.textContent = '\\u2713 Auto-filled with ' + ticketId; hintWs.classList.add('visible'); }

    UI.toast('All test fields updated with ticket ' + ticketId, 'success');
  },

  createTicket: async function() {
    var amount = parseFloat(document.getElementById('tc-amount').value) || 500;
    var res = await API.post('/api/ticket', { amount: amount }, {});
    this.showResponse('resp-create', res);
    if (res.ok && res.data.ticketId) {
      this.autoFill(res.data.ticketId, res.data.amount || amount);
    }
  },

  smsWebhook: async function() {
    var secret = document.getElementById('tw-secret').value;
    var sms = document.getElementById('tw-sms').value;
    var res = await API.post('/api/webhook', { sms: sms }, { 'X-Webhook-Secret': secret });
    this.showResponse('resp-sms', res);
  },

  kotakWebhook: async function() {
    var secret = document.getElementById('tk-secret').value;
    var sms = document.getElementById('tk-sms').value;
    var res = await API.post('/api/webhook', { sms: sms }, { 'X-Webhook-Secret': secret });
    this.showResponse('resp-kotak', res);
  },

  emailWebhook: async function() {
    var secret = document.getElementById('te-secret').value;
    var from = document.getElementById('te-from').value;
    var subject = document.getElementById('te-subject').value;
    var text = document.getElementById('te-body').value;
    var res = await API.post('/api/email-webhook', { from: from, subject: subject, text: text }, { 'X-Email-Secret': secret });
    this.showResponse('resp-email', res);
  },

  checkStatus: async function() {
    var id = document.getElementById('ts-id').value.trim();
    if (!id) { UI.toast('Enter a ticket ID', 'warn'); return; }
    var res = await API.get('/api/status/' + encodeURIComponent(id));
    this.showResponse('resp-status', res);
  },

  togglePingWs: function() {
    var btn = document.getElementById('ws-ping-btn');
    if (State.pingWs) {
      State.pingWs.close();
      State.pingWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
      return;
    }
    var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    var ws = new WebSocket(proto + '//' + location.host + '/api/ping');
    State.pingWs = ws;
    btn.textContent = 'Connecting...';
    ws.onopen = function() {
      AppConsole.log('ok', '[WS-Ping] Connected');
      btn.textContent = 'Disconnect';
      btn.classList.add('connected');
      var respEl = document.getElementById('resp-ping');
      respEl.innerHTML = '<div class="test-response-head"><span class="status-pill status-ok">OPEN</span></div><div class="test-response-body">Connected. Waiting for pong...</div>';
      respEl.classList.add('visible');
      ws.send('ping');
    };
    ws.onmessage = function(e) {
      AppConsole.log('info', '[WS-Ping] Received: ' + e.data);
      var respEl = document.getElementById('resp-ping');
      if (respEl) {
        var body = respEl.querySelector('.test-response-body');
        if (body) body.textContent = 'Received: ' + e.data;
      }
    };
    ws.onclose = function() {
      AppConsole.log('warn', '[WS-Ping] Closed');
      State.pingWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
    };
    ws.onerror = function() {
      AppConsole.log('error', '[WS-Ping] Error');
      State.pingWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
    };
  },

  toggleTicketWs: function() {
    var btn = document.getElementById('ws-ticket-btn');
    if (State.ticketWs) {
      State.ticketWs.close();
      State.ticketWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
      return;
    }
    var ticketId = document.getElementById('ws-ticket-id').value.trim();
    if (!ticketId) { UI.toast('Enter a ticket ID', 'warn'); return; }
    var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    var ws = new WebSocket(proto + '//' + location.host + '/api/ws?ticketId=' + encodeURIComponent(ticketId));
    State.ticketWs = ws;
    btn.textContent = 'Connecting...';
    ws.onopen = function() {
      AppConsole.log('ok', '[WS-Ticket] Connected for ' + ticketId);
      btn.textContent = 'Disconnect';
      btn.classList.add('connected');
      var respEl = document.getElementById('resp-ticketws');
      respEl.innerHTML = '<div class="test-response-head"><span class="status-pill status-ok">OPEN</span></div><div class="test-response-body">Listening for payment updates on ' + Util.escape(ticketId) + '...</div>';
      respEl.classList.add('visible');
    };
    ws.onmessage = function(e) {
      AppConsole.log('info', '[WS-Ticket] ' + e.data);
      var respEl = document.getElementById('resp-ticketws');
      if (respEl) {
        try {
          var d = JSON.parse(e.data);
          var body = respEl.querySelector('.test-response-body');
          if (body) body.innerHTML = Util.colorizeJson(d);
          if (d.status === 'paid') UI.toast('Payment confirmed! ' + ticketId, 'success');
        } catch(_) {}
      }
    };
    ws.onclose = function() {
      AppConsole.log('warn', '[WS-Ticket] Closed');
      State.ticketWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
    };
    ws.onerror = function() {
      AppConsole.log('error', '[WS-Ticket] Error');
      State.ticketWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
    };
  }
};

// \u2500\u2500\u2500 CONSOLE \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
var AppConsole = {
  setFilter: function(level) {
    State.logFilter = level;
    document.querySelectorAll('.console-level-btn').forEach(function(el) {
      el.classList.toggle('active', el.dataset.level === level);
    });
    this.rerender();
  },
  log: function(level, msg, data) {
    var now = new Date();
    var time = now.toTimeString().split(' ')[0] + '.' + String(now.getMilliseconds()).padStart(3,'0');
    State.logs.push({ level: level, msg: msg, data: data || null, time: time });
    if (State.logs.length > 500) State.logs.shift();
    if (!State.logFilter || State.logFilter === level) this.appendEntry(State.logs[State.logs.length-1]);
  },
  appendEntry: function(entry) {
    var box = document.getElementById('console-box');
    if (!box) return;
    var div = document.createElement('div');
    div.className = 'log-entry';
    div.innerHTML = '<span class="log-time">' + Util.escape(entry.time) + '</span>'
      + '<span class="log-level log-level-' + entry.level + '">' + entry.level.toUpperCase() + '</span>'
      + '<span class="log-msg">' + Util.escape(entry.msg) + '</span>';
    box.appendChild(div);
    box.scrollTop = box.scrollHeight;
  },
  rerender: function() {
    var box = document.getElementById('console-box');
    if (!box) return;
    box.innerHTML = '';
    var filter = State.logFilter;
    for (var i = 0; i < State.logs.length; i++) {
      if (!filter || State.logs[i].level === filter) this.appendEntry(State.logs[i]);
    }
    box.scrollTop = box.scrollHeight;
  },
  clear: function() {
    State.logs = [];
    var box = document.getElementById('console-box');
    if (box) box.innerHTML = '';
  }
};

// \u2500\u2500\u2500 UI \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
var UI = {
  showTableLoading: function() {
    var tbody = document.getElementById('ticket-tbody');
    if (!tbody) return;
    var html = '';
    for (var i = 0; i < 8; i++) {
      html += '<tr><td><div class="skeleton" style="width:120px"></div></td>'
        + '<td><div class="skeleton" style="width:60px"></div></td>'
        + '<td><div class="skeleton" style="width:70px"></div></td>'
        + '<td><div class="skeleton" style="width:100px"></div></td>'
        + '<td><div class="skeleton" style="width:120px"></div></td>'
        + '<td><div class="skeleton" style="width:80px"></div></td>'
        + '<td><div class="skeleton" style="width:70px"></div></td>'
        + '<td><div class="skeleton" style="width:80px"></div></td></tr>';
    }
    tbody.innerHTML = html;
  },
  animateCounter: function(el, from, to) {
    var start = null;
    var dur = 400;
    var step = function(ts) {
      if (!start) start = ts;
      var prog = Math.min((ts - start) / dur, 1);
      var ease = 1 - Math.pow(1 - prog, 3);
      el.textContent = Math.round(from + (to - from) * ease);
      if (prog < 1) requestAnimationFrame(step);
      else el.textContent = to;
    };
    requestAnimationFrame(step);
  },
  toast: function(msg, type) {
    var stack = document.getElementById('toast-stack');
    var el = document.createElement('div');
    el.className = 'toast toast-' + (type || 'info');
    el.textContent = msg;
    stack.appendChild(el);
    setTimeout(function() {
      el.style.animation = 'toastOut 0.25s var(--ease-out) forwards';
      setTimeout(function() { if (el.parentNode) el.parentNode.removeChild(el); }, 280);
    }, 3000);
  }
};

// \u2500\u2500\u2500 TABS \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
function switchTab(tab) {
  State.activeTab = tab;
  document.querySelectorAll('.nav-tab').forEach(function(el) {
    el.classList.toggle('active', el.dataset.tab === tab);
  });
  var tabs = ['overview','testing','console'];
  for (var i = 0; i < tabs.length; i++) {
    var el = document.getElementById('tab-' + tabs[i]);
    if (el) el.style.display = (tabs[i] === tab) ? 'block' : 'none';
  }
}

// \u2500\u2500\u2500 EVENT WIRING \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500
function init() {
  // Tab switching
  document.querySelectorAll('.nav-tab').forEach(function(el) {
    el.addEventListener('click', function() { switchTab(el.dataset.tab); });
  });

  // Refresh button
  document.getElementById('refresh-btn').addEventListener('click', function() {
    Dashboard.loadTickets(true, true);
  });

  // Sign out
  document.getElementById('signout-btn').addEventListener('click', function() {
    Auth.logout();
  });

  // Search \u2014 debounced DB call
  document.getElementById('search-input').addEventListener('input', function() {
    State.search = this.value;
    clearTimeout(State.searchTimer);
    State.searchTimer = setTimeout(function() {
      Dashboard.loadTickets(true, true);
    }, 450);
  });

  // Date filters
  document.getElementById('date-from').addEventListener('change', function() {
    State.dateFrom = this.value;
    Dashboard.loadTickets(true, true);
  });
  document.getElementById('date-to').addEventListener('change', function() {
    State.dateTo = this.value;
    Dashboard.loadTickets(true, true);
  });
  document.getElementById('date-clear').addEventListener('click', function() {
    State.dateFrom = ''; State.dateTo = '';
    document.getElementById('date-from').value = '';
    document.getElementById('date-to').value = '';
    Dashboard.loadTickets(true, true);
  });

  // Close modal on overlay click
  document.getElementById('modal-overlay').addEventListener('click', function(e) {
    if (e.target === this) Modal.close();
  });

  // Close modal on Escape
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && document.getElementById('modal-overlay').style.display !== 'none') {
      Modal.close();
    }
  });

  Auth.init();
}

init();

// expose for inline onclick handlers
window.Dashboard = Dashboard;
window.Modal = Modal;
window.Testing = Testing;
window.AppConsole = AppConsole;

})();
<\/script>
</body>
</html>`;

// src/lib/emailParser.ts
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_process();
init_virtual_unenv_global_polyfill_cloudflare_unenv_preset_node_console();
init_performance2();
var EmailParser = class {
  static {
    __name(this, "EmailParser");
  }
  /**
   * Extracts the payment amount, decimal parts, RRN, and Sender Name from a Slice UPI email.
   */
  static parseSliceEmail(subject, textOrHtmlBody) {
    const result = {
      paidAmount: null,
      intPart: null,
      decPart: null,
      rrn: null,
      senderName: "UNKNOWN"
    };
    const normalised = textOrHtmlBody.replace(/=\r?\n/g, "").replace(/=3D/g, "=").replace(/=E2=82=B9/gi, "\u20B9").replace(/&nbsp;/g, " ");
    let amountMatch = subject.match(/₹\s*(\d+)(?:\.(\d{2}))?/i);
    if (!amountMatch) {
      amountMatch = normalised.match(/received\s*₹\s*(\d+)(?:\.(\d{2}))?\s*via/i) || normalised.match(/₹\s*(\d+)(?:\.(\d{2}))?/);
    }
    if (amountMatch) {
      result.intPart = parseInt(amountMatch[1], 10);
      result.decPart = amountMatch[2] ? parseInt(amountMatch[2], 10) : 0;
      result.paidAmount = parseFloat(
        `${result.intPart}.${amountMatch[2] || "00"}`
      );
    }
    const rrnMatch = normalised.match(/RRN\s*<\/td>\s*<td[^>]*>\s*(\d{9,15})/i) || normalised.match(/RRN\D{0,10}(\d{9,15})/i);
    if (rrnMatch) {
      result.rrn = rrnMatch[1];
    }
    const senderMatch = normalised.match(/From\s*<\/td>\s*<td[^>]*>\s*([A-Z0-9 ]+)\s*</i) || normalised.match(/From\s*[\|\t:]\s*([A-Z0-9 ]+)/i);
    if (senderMatch) {
      result.senderName = senderMatch[1].trim();
    }
    return result;
  }
  /**
   * Extracts the payment amount, decimal parts, RRN, and Sender Name from a Kotak SMS.
   * Example: "Confirmed payment for Received Rs.3.00 in your Kotak Bank AC X4959 from drvijayapalliyil@oksbi on 08-03-26.UPI Ref:606703736479."
   */
  static parseKotakSms(smsBody) {
    if (!smsBody.toUpperCase().includes("KOTAK")) {
      return null;
    }
    const result = {
      paidAmount: null,
      intPart: null,
      decPart: null,
      rrn: null,
      senderName: "UNKNOWN",
      upiId: null
    };
    const amountMatch = smsBody.match(/Rs\.?\s*(\d+)(?:\.(\d{2}))?/i);
    if (amountMatch) {
      result.intPart = parseInt(amountMatch[1], 10);
      result.decPart = amountMatch[2] ? parseInt(amountMatch[2], 10) : 0;
      result.paidAmount = parseFloat(`${result.intPart}.${amountMatch[2] || "00"}`);
    } else {
      return null;
    }
    const upiIdMatch = smsBody.match(/from\s+([a-zA-Z0-9.\-_]+@[a-zA-Z0-9.\-_]+)/i);
    if (upiIdMatch) {
      result.upiId = upiIdMatch[1].trim();
    }
    const rrnMatch = smsBody.match(/UPI Ref[:\s]*(\d+)/i) || smsBody.match(/Ref\.?No\.?[:\s]*(\d+)/i) || smsBody.match(/RRN[:\s]*(\d+)/i);
    if (rrnMatch) {
      result.rrn = rrnMatch[1];
    }
    return result;
  }
};

// src/worker.ts
var localDecimalLocks = /* @__PURE__ */ new Set();
function timingSafeEqual(a3, b) {
  const aLen = a3.length;
  const bLen = b.length;
  let result = aLen ^ bLen;
  const len = Math.min(aLen, bLen);
  for (let i3 = 0; i3 < len; i3++) {
    result |= a3.charCodeAt(i3) ^ b.charCodeAt(i3);
  }
  return result === 0;
}
__name(timingSafeEqual, "timingSafeEqual");
async function requireAdminAuth(c, next) {
  const password = c.req.header("X-Admin-Password") || "";
  const adminPassword = c.env.ADMIN_PASSWORD || "admin123";
  if (!timingSafeEqual(password, adminPassword)) {
    return c.json({ error: "Unauthorized" }, 401);
  }
  return next();
}
__name(requireAdminAuth, "requireAdminAuth");
var app = new Hono2();
app.use("/*", async (c, next) => {
  const allowedOrigin = c.env.ALLOWED_ORIGIN || "*";
  const corsMiddleware = cors({
    origin: allowedOrigin.includes(",") ? allowedOrigin.split(",") : allowedOrigin
  });
  return corsMiddleware(c, next);
});
app.get("/", (c) => {
  return c.text("Payment Gateway API is running!");
});
app.get(
  "/api/ping",
  upgradeWebSocket((c) => {
    return {
      onMessage(event, ws) {
        console.log("PING RX");
        ws.send("pong");
      },
      onOpen() {
        console.log("PING WS OPEN");
      }
    };
  })
);
app.post("/api/ticket", async (c) => {
  try {
    const body = await c.req.json();
    const baseAmount = body.amount;
    if (!baseAmount || typeof baseAmount !== "number") {
      return c.json({ error: "Invalid amount" }, 400);
    }
    const appwrite = new AppwriteService(c.env);
    const dbAllocatedDecimals = await appwrite.getPendingDecimalsForAmount(
      Math.floor(baseAmount)
    );
    let availableDecimal = -1;
    const startOffset = Math.floor(Math.random() * 100);
    for (let i3 = 0; i3 < 100; i3++) {
      const candidateDecimal = (startOffset + i3) % 100;
      const lockKey = `${Math.floor(baseAmount)}_${candidateDecimal}`;
      if (!dbAllocatedDecimals.includes(candidateDecimal) && !localDecimalLocks.has(lockKey)) {
        const lockAcquired = await appwrite.claimDatabaseLock(
          Math.floor(baseAmount),
          candidateDecimal
        );
        if (lockAcquired) {
          availableDecimal = candidateDecimal;
          localDecimalLocks.add(lockKey);
          setTimeout(() => localDecimalLocks.delete(lockKey), 5 * 60 * 1e3);
          break;
        }
      }
    }
    if (availableDecimal === -1) {
      return c.json(
        {
          error: "System busy: Too many concurrent transactions for this amount. Please try again later."
        },
        503
      );
    }
    const finalAmount = Math.floor(baseAmount) + availableDecimal / 100;
    const timestamp = Date.now().toString();
    const prefix = timestamp.slice(0, -2);
    const decimalStr = availableDecimal.toString().padStart(2, "0");
    const ticketId = `TICKET${prefix}${decimalStr}`;
    const createdDoc = await appwrite.createTicket(ticketId, finalAmount);
    return c.json({
      ticketId,
      // THE READABLE TICKET REFERENCE
      amount: finalAmount,
      status: "pending",
      createdAt: createdDoc.$createdAt
    });
  } catch (error3) {
    console.error("Error creating ticket:", error3);
    return c.json({ error: "Internal Server Error" }, 500);
  }
});
app.post("/api/webhook", async (c) => {
  try {
    const rawBody = await c.req.text();
    console.log("--- WEBHOOK RECEIVED ---");
    console.log("TIMESTAMP:", (/* @__PURE__ */ new Date()).toISOString());
    console.log("RAW BODY:", rawBody);
    let body;
    try {
      body = JSON.parse(rawBody);
    } catch (e3) {
      console.log(
        "Could not parse JSON body, assuming raw text or URL encoded."
      );
      body = { sms: rawBody };
    }
    const secret = c.req.header("X-Webhook-Secret") || c.req.query("secret") || body.secret_key;
    if (!secret || !timingSafeEqual(secret, c.env.WEBHOOK_SECRET)) {
      console.log("Unauthorized Webhook Attempt");
      return c.json({ error: "Unauthorized" }, 401);
    }
    const content = ((body.sms || "") + " " + (body.body || "") + " " + (body.message || "")).trim() || rawBody;
    console.log("SEARCH CONTENT:", content);
    const ticketMatch = content.match(/TICKET(\d+)/);
    const paymentSource = body.body || content;
    const paymentMatch = paymentSource.match(
      /([a-zA-Z0-9\s\.]+?) (?:has )?paid you ₹(\d+(\.\d{1,2})?)/i
    );
    let foundId = null;
    let status = "ignored";
    let updatedDoc = null;
    if (ticketMatch && paymentMatch) {
      foundId = ticketMatch[0];
      const senderName = paymentMatch[1].trim();
      const paidAmount = parseFloat(paymentMatch[2]);
      console.log("FOUND TICKET ID:", foundId);
      console.log("SENDER:", senderName);
      console.log("PAID AMOUNT:", paidAmount);
      const appwrite = new AppwriteService(c.env);
      const ticket = await appwrite.getTicketStatus(foundId);
      if (!ticket) {
        console.log(`Ticket ${foundId} NOT FOUND`);
        status = "ticket_not_found";
      } else if (toCents(ticket.amount) !== toCents(paidAmount)) {
        console.log(
          `AMOUNT MISMATCH: Ticket requires ${ticket.amount}, but received ${paidAmount} `
        );
        status = "amount_mismatch";
      } else {
        if (ticket.status === "paid") {
          console.log(`Ticket ${foundId} ALREADY PAID, updating sender name anyway`);
        }
        updatedDoc = await appwrite.markAsPaid(foundId, senderName);
        if (updatedDoc) {
          console.log(`Ticket ${foundId} MARKED AS PAID`);
          status = "success";
          const baseAmount = Math.floor(ticket.amount);
          const decPart = Math.round((ticket.amount - baseAmount) * 100);
          localDecimalLocks.delete(`${baseAmount}_${decPart}`);
          c.executionCtx.waitUntil(
            appwrite.releaseDatabaseLock(baseAmount, decPart).catch(() => null)
          );
        } else {
          status = "update_failed";
        }
      }
    } else {
      const kotakParsed = EmailParser.parseKotakSms(content);
      if (kotakParsed && kotakParsed.paidAmount !== null && kotakParsed.intPart !== null && kotakParsed.decPart !== null) {
        console.log("KOTAK SMS DETECTED WITHOUT EXPLICIT TICKET ID");
        const { paidAmount, decPart, intPart, rrn, upiId } = kotakParsed;
        console.log("PAID AMOUNT:", paidAmount, "| DEC PART:", decPart);
        console.log("UPI ID:", upiId, "| RRN:", rrn);
        const appwrite = new AppwriteService(c.env);
        const candidates = await appwrite.listRecentTickets(20);
        const now = Date.now();
        const FIVE_MIN_MS = 5 * 60 * 1e3;
        let matchedTicket = null;
        let matchedTicketId = null;
        for (const ticket of candidates) {
          if (ticket.ticketId.startsWith("lock_")) continue;
          const ticketTime = new Date(ticket.createdAt).getTime();
          if (now - ticketTime > FIVE_MIN_MS) continue;
          const numericPart = ticket.ticketId.replace(/^TICKET/i, "");
          const ticketSuffix = parseInt(numericPart.slice(-2), 10);
          if (ticketSuffix !== decPart) continue;
          if (toCents(Math.floor(ticket.amount)) !== toCents(intPart)) continue;
          matchedTicket = ticket;
          matchedTicketId = ticket.ticketId;
          break;
        }
        if (matchedTicket && matchedTicketId) {
          updatedDoc = await appwrite.markAsPaid(matchedTicketId, void 0, rrn ?? void 0, upiId ?? void 0);
          if (updatedDoc) {
            console.log(`Ticket ${matchedTicketId} MARKED AS PAID via Kotak SMS. RRN: ${rrn}`);
            status = "success";
            foundId = matchedTicketId;
            localDecimalLocks.delete(`${intPart}_${decPart}`);
            c.executionCtx.waitUntil(
              appwrite.releaseDatabaseLock(intPart, decPart).catch(() => null)
            );
          } else {
            status = "update_failed";
          }
        } else {
          console.log(`No matching pending ticket found for Kotak amount \u20B9${paidAmount} (dec = ${decPart})`);
          status = "no_matching_ticket";
        }
      } else {
        console.log("INVALID SMS FORMAT: Missing Ticket ID or Payment Details");
        if (!ticketMatch) console.log(" - Missing Ticket ID");
        if (!paymentMatch)
          console.log(" - Missing Payment Details (Name/Amount)");
        status = "invalid_format";
      }
    }
    return c.json({
      status: "received",
      processed_id: foundId,
      action: status
    });
  } catch (error3) {
    console.error("Webhook error:", error3);
    return c.json({ error: "Internal Server Error" }, 500);
  }
});
app.post("/api/email-webhook", async (c) => {
  try {
    const rawBody = await c.req.text();
    console.log("--- EMAIL WEBHOOK RECEIVED ---");
    console.log("TIMESTAMP:", (/* @__PURE__ */ new Date()).toISOString());
    let body;
    try {
      body = JSON.parse(rawBody);
    } catch {
      return c.json({ error: "Invalid JSON" }, 400);
    }
    const secret = c.req.header("X-Email-Secret") || c.req.query("secret") || body.secret;
    if (!secret || !timingSafeEqual(secret, c.env.EMAIL_SECRET)) {
      console.log("Unauthorized email webhook attempt");
      return c.json({ error: "Unauthorized" }, 401);
    }
    const from = (body.from || "").toLowerCase().trim();
    const subject = (body.subject || "").trim();
    const text = (body.text || body.html || "").replace(/=\r?\n/g, "").replace(/=3D/g, "=");
    console.log("FROM:", from, "| SUBJECT:", subject);
    if (!subject.toLowerCase().includes("received") || !subject.toLowerCase().includes("via upi")) {
      console.log("Rejected: subject does not match UPI credit pattern");
      return c.json({ status: "ignored", reason: "subject_mismatch" });
    }
    const parsedEmail = EmailParser.parseSliceEmail(subject, text);
    if (parsedEmail.paidAmount === null || parsedEmail.decPart === null || parsedEmail.intPart === null) {
      console.log("Could not extract amount from email body or subject");
      return c.json({ status: "ignored", reason: "amount_not_found" });
    }
    const { paidAmount, decPart, intPart, rrn, senderName } = parsedEmail;
    console.log("PAID AMOUNT:", paidAmount, "| DEC PART:", decPart);
    console.log("RRN:", rrn);
    console.log("SENDER NAME:", senderName);
    const appwrite = new AppwriteService(c.env);
    const candidates = await appwrite.listRecentPendingTickets(20);
    const now = Date.now();
    const FIVE_MIN_MS = 5 * 60 * 1e3;
    let matchedTicket = null;
    let matchedTicketId = null;
    for (const ticket of candidates) {
      if (ticket.ticketId.startsWith("lock_")) continue;
      const ticketTime = new Date(ticket.createdAt).getTime();
      if (now - ticketTime > FIVE_MIN_MS) {
        console.log(
          `Ticket ${ticket.ticketId}: outside 5 - minute window, skipping`
        );
        continue;
      }
      const numericPart = ticket.ticketId.replace(/^TICKET/i, "");
      const ticketSuffix = parseInt(numericPart.slice(-2), 10);
      if (ticketSuffix !== decPart) {
        console.log(
          `Ticket ${ticket.ticketId}: suffix ${ticketSuffix} !== dec ${decPart}, skipping`
        );
        continue;
      }
      if (toCents(Math.floor(ticket.amount)) !== toCents(intPart)) {
        console.log(
          `Ticket ${ticket.ticketId}: integer amount mismatch(expected ${ticket.amount}, got ${intPart}), skipping`
        );
        continue;
      }
      if (ticket.status === "paid") {
        console.log(`Ticket ${ticket.ticketId}: already paid, skipping`);
        continue;
      }
      matchedTicket = ticket;
      matchedTicketId = ticket.ticketId;
      break;
    }
    if (!matchedTicket || !matchedTicketId) {
      console.log(
        `No matching pending ticket found for amount \u20B9${paidAmount} (dec = ${decPart})`
      );
      return c.json({
        status: "ignored",
        reason: "no_matching_ticket",
        paid_amount: paidAmount,
        dec_part: decPart
      });
    }
    const updatedDoc = await appwrite.markAsPaid(
      matchedTicketId,
      senderName,
      rrn ?? void 0
    );
    if (updatedDoc) {
      console.log(
        `Ticket ${matchedTicketId} MARKED AS PAID via email. RRN: ${rrn}`
      );
      localDecimalLocks.delete(`${intPart}_${decPart}`);
      c.executionCtx.waitUntil(
        appwrite.releaseDatabaseLock(intPart, decPart).catch(() => null)
      );
      const { id, ...sanitized } = updatedDoc;
      return c.json(sanitized);
    } else {
      return c.json({ status: "error", reason: "update_failed" }, 500);
    }
  } catch (error3) {
    console.error("Email webhook error:", error3);
    return c.json({ error: "Internal Server Error" }, 500);
  }
});
app.get("/api/status/:id", async (c) => {
  const id = c.req.param("id");
  const appwrite = new AppwriteService(c.env);
  const status = await appwrite.getTicketStatus(id);
  if (!status) {
    return c.json({ status: "not_found" }, 404);
  }
  const { id: internalId, ...sanitized } = status;
  return c.json(sanitized);
});
app.get("/api/ws", async (c) => {
  const ticketId = (c.req.query("ticketId") || "").trim();
  if (!ticketId || !/^TICKET\d+$/.test(ticketId)) {
    return c.text("Invalid ticketId", 400);
  }
  const pair = new WebSocketPair();
  const client = pair[0];
  const server = pair[1];
  server.accept();
  let upstreamWs = null;
  let closing = false;
  const safeCloseAll = /* @__PURE__ */ __name(() => {
    if (closing) return;
    closing = true;
    try {
      server.close();
    } catch {
    }
    try {
      upstreamWs?.close();
    } catch {
    }
  }, "safeCloseAll");
  server.addEventListener("close", safeCloseAll);
  server.addEventListener("error", safeCloseAll);
  server.addEventListener("message", (event) => {
    try {
      if (event.data === "ping") server.send("pong");
    } catch {
    }
  });
  const appwrite = new AppwriteService(c.env);
  const snapshotPromise = appwrite.getTicketStatus(ticketId).then((status) => {
    if (!status) return;
    try {
      server.send(
        JSON.stringify({
          type: "payment_update",
          status: status.status,
          paidAt: status.paidAt || null
        })
      );
    } catch {
    }
  }).catch(() => null);
  c.executionCtx?.waitUntil(snapshotPromise);
  const appwriteHost = c.env.APPWRITE_ENDPOINT.replace(/^https?:\/\//, "").split("/")[0];
  const channel2 = `databases.${c.env.APPWRITE_DATABASE_ID}.collections.${c.env.APPWRITE_COLLECTION_ID}.documents.${ticketId}`;
  const appwriteWsUrl = `wss://${appwriteHost}/v1/realtime?project=${c.env.APPWRITE_PROJECT_ID}&channels[]=${encodeURIComponent(
    channel2
  )}`;
  const upstreamReady = fetch(appwriteWsUrl.replace("wss://", "https://"), {
    headers: {
      Upgrade: "websocket",
      "X-Appwrite-Project": c.env.APPWRITE_PROJECT_ID,
      "X-Appwrite-Key": c.env.APPWRITE_API_KEY
    }
  }).then((res) => {
    const ws = res.webSocket;
    if (!ws) throw new Error("Appwrite handshake failed");
    ws.accept();
    upstreamWs = ws;
    return ws;
  }).then((ws) => {
    ws.addEventListener("message", (msg) => {
      try {
        const envelope = JSON.parse(msg.data);
        if (envelope?.type !== "event") return;
        const doc = envelope?.data?.payload;
        if (!doc) return;
        const docId = doc.$id || doc.ticketId;
        const status = doc.status;
        if (docId !== ticketId || !status) return;
        server.send(
          JSON.stringify({
            type: "payment_update",
            status,
            paidAt: doc.paidAt || null
          })
        );
      } catch {
      }
    });
    ws.addEventListener("close", safeCloseAll);
    ws.addEventListener("error", safeCloseAll);
  }).catch((err) => {
    console.error(`[WS-PROXY] Upstream fatal: ${err}`);
    safeCloseAll();
  });
  c.executionCtx?.waitUntil(upstreamReady);
  return new Response(null, { status: 101, webSocket: client });
});
app.get("/admin", (c) => {
  return c.html(ADMIN_HTML);
});
app.get("/api/admin/tickets", requireAdminAuth, async (c) => {
  try {
    const page = Math.max(0, parseInt(c.req.query("page") || "0"));
    const pageSize = Math.min(Math.max(1, parseInt(c.req.query("pageSize") || "10")), 100);
    const search = (c.req.query("search") || "").toLowerCase().trim();
    const statusFilter = c.req.query("status") || "";
    const dateFrom = c.req.query("dateFrom") || "";
    const dateTo = c.req.query("dateTo") || "";
    const isoFrom = dateFrom ? (/* @__PURE__ */ new Date(dateFrom + "T00:00:00.000Z")).toISOString() : void 0;
    const isoTo = dateTo ? (/* @__PURE__ */ new Date(dateTo + "T23:59:59.999Z")).toISOString() : void 0;
    const appwrite = new AppwriteService(c.env);
    let tickets = await appwrite.listAllTickets({
      statusFilter: statusFilter || void 0,
      dateFrom: isoFrom,
      dateTo: isoTo
    });
    if (search) {
      tickets = tickets.filter(
        (t2) => t2.ticketId.toLowerCase().includes(search) || (t2.senderName || "").toLowerCase().includes(search) || (t2.upiId || "").toLowerCase().includes(search) || (t2.rrn || "").toLowerCase().includes(search) || t2.amount.toString().includes(search)
      );
    }
    const total = tickets.length;
    const paid = tickets.filter((t2) => t2.status === "paid").length;
    const pending = tickets.filter((t2) => t2.status === "pending").length;
    const cancelled = tickets.filter((t2) => t2.status === "cancelled").length;
    const start = page * pageSize;
    const pageSlice = tickets.slice(start, start + pageSize);
    const hasMore = start + pageSize < total;
    return c.json({
      tickets: pageSlice,
      stats: { total, paid, pending, cancelled },
      hasMore,
      page,
      pageSize
    });
  } catch (error3) {
    console.error("Admin list tickets error:", error3);
    return c.json({ error: "Internal Server Error" }, 500);
  }
});
app.get("/api/admin/ws", async (c) => {
  const pw = c.req.query("pw") || "";
  const adminPassword = c.env.ADMIN_PASSWORD || "admin123";
  if (!timingSafeEqual(pw, adminPassword)) {
    return c.text("Unauthorized", 401);
  }
  const pair = new WebSocketPair();
  const client = pair[0];
  const server = pair[1];
  server.accept();
  let upstreamWs = null;
  let closing = false;
  const safeCloseAll = /* @__PURE__ */ __name(() => {
    if (closing) return;
    closing = true;
    try {
      server.close();
    } catch {
    }
    try {
      upstreamWs?.close();
    } catch {
    }
  }, "safeCloseAll");
  server.addEventListener("close", safeCloseAll);
  server.addEventListener("error", safeCloseAll);
  server.addEventListener("message", (event) => {
    try {
      if (event.data === "ping") server.send("pong");
    } catch {
    }
  });
  const appwriteHost = c.env.APPWRITE_ENDPOINT.replace(/^https?:\/\//, "").split("/")[0];
  const channel2 = `databases.${c.env.APPWRITE_DATABASE_ID}.collections.${c.env.APPWRITE_COLLECTION_ID}.documents`;
  const appwriteWsUrl = `wss://${appwriteHost}/v1/realtime?project=${c.env.APPWRITE_PROJECT_ID}&channels[]=${encodeURIComponent(channel2)}`;
  const upstreamReady = fetch(appwriteWsUrl.replace("wss://", "https://"), {
    headers: {
      Upgrade: "websocket",
      "X-Appwrite-Project": c.env.APPWRITE_PROJECT_ID,
      "X-Appwrite-Key": c.env.APPWRITE_API_KEY
    }
  }).then((res) => {
    const ws = res.webSocket;
    if (!ws) throw new Error("Appwrite admin WS handshake failed");
    ws.accept();
    upstreamWs = ws;
    return ws;
  }).then((ws) => {
    ws.addEventListener("message", (msg) => {
      try {
        const envelope = JSON.parse(msg.data);
        if (envelope?.type !== "event") return;
        const doc = envelope?.data?.payload;
        if (!doc) return;
        if (doc.ticketId?.startsWith("lock_")) return;
        const events = envelope?.data?.events || [];
        let action = "update";
        if (events.some((e3) => e3.endsWith(".create"))) action = "create";
        else if (events.some((e3) => e3.endsWith(".delete"))) action = "delete";
        server.send(JSON.stringify({
          type: "ticket_update",
          action,
          ticket: {
            id: doc.$id,
            ticketId: doc.ticketId,
            amount: doc.amount,
            status: doc.status,
            createdAt: doc.$createdAt,
            senderName: doc.senderName ?? null,
            rrn: doc.rrn ?? null,
            paidAt: doc.paidAt ?? null,
            upiId: doc.upiId ?? null
          }
        }));
      } catch {
      }
    });
    ws.addEventListener("close", safeCloseAll);
    ws.addEventListener("error", safeCloseAll);
  }).catch((err) => {
    console.error(`[ADMIN-WS] Upstream fatal: ${err}`);
    safeCloseAll();
  });
  c.executionCtx?.waitUntil(upstreamReady);
  return new Response(null, { status: 101, webSocket: client });
});
app.post("/api/admin/tickets/:id/mark-paid", requireAdminAuth, async (c) => {
  try {
    const id = c.req.param("id");
    const body = await c.req.json().catch(() => ({}));
    const fields = {
      status: "paid",
      paidAt: (/* @__PURE__ */ new Date()).toISOString()
    };
    if (body.senderName) fields.senderName = body.senderName;
    if (body.rrn) fields.rrn = body.rrn;
    if (body.upiId) fields.upiId = body.upiId;
    const appwrite = new AppwriteService(c.env);
    const updated = await appwrite.updateTicket(id, fields);
    if (!updated) return c.json({ error: "Update failed" }, 500);
    return c.json(updated);
  } catch (error3) {
    console.error("Admin mark-paid error:", error3);
    return c.json({ error: "Internal Server Error" }, 500);
  }
});
app.post("/api/admin/tickets/:id/cancel", requireAdminAuth, async (c) => {
  try {
    const id = c.req.param("id");
    const appwrite = new AppwriteService(c.env);
    const updated = await appwrite.updateTicket(id, { status: "cancelled" });
    if (!updated) return c.json({ error: "Update failed" }, 500);
    return c.json(updated);
  } catch (error3) {
    console.error("Admin cancel error:", error3);
    return c.json({ error: "Internal Server Error" }, 500);
  }
});
app.get("/api/admin/tickets/:id", requireAdminAuth, async (c) => {
  try {
    const id = c.req.param("id");
    const appwrite = new AppwriteService(c.env);
    const ticket = await appwrite.getTicketStatus(id);
    if (!ticket) return c.json({ error: "Not found" }, 404);
    return c.json(ticket);
  } catch (error3) {
    console.error("Admin get ticket error:", error3);
    return c.json({ error: "Internal Server Error" }, 500);
  }
});
app.patch("/api/admin/tickets/:id", requireAdminAuth, async (c) => {
  try {
    const id = c.req.param("id");
    let body;
    try {
      body = await c.req.json();
    } catch {
      return c.json({ error: "Invalid JSON body" }, 400);
    }
    const allowedKeys = ["status", "senderName", "rrn", "upiId", "amount", "paidAt"];
    const fields = {};
    for (const key of allowedKeys) {
      if (key in body) fields[key] = body[key];
    }
    const appwrite = new AppwriteService(c.env);
    const updated = await appwrite.updateTicket(id, fields);
    if (!updated) return c.json({ error: "Update failed or no fields provided" }, 400);
    return c.json(updated);
  } catch (error3) {
    console.error("Admin patch ticket error:", error3);
    return c.json({ error: "Internal Server Error" }, 500);
  }
});
var worker_default = {
  fetch: app.fetch,
  async email(message, env2, ctx) {
    try {
      console.log("Received email from:", message.from, "to:", message.to);
      const rawEmail = await new Response(message.raw).text();
      const payload = {
        secret: env2.EMAIL_SECRET,
        from: message.from,
        subject: message.headers.get("Subject") || "",
        text: rawEmail
      };
      const req = new Request("http://localhost/api/email-webhook", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      });
      const res = await app.fetch(req, env2, ctx);
      console.log("Email webhook local processing status:", res.status);
      const resultText = await res.text();
      console.log("Email webhook result:", resultText);
    } catch (e3) {
      console.error("Email worker error:", e3);
    }
  }
};
export {
  worker_default as default
};
//# sourceMappingURL=worker.js.map
