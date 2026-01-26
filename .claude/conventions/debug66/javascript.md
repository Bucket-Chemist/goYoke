---
description: Debug 66 JavaScript/TypeScript implementation. Verbose step-trace debugging using console. Includes extensions for Bitburner NS API, async operations, and game-specific patterns.
globs: ["*.js", "*.ts", "*.mjs", "*.jsx", "*.tsx"]
alwaysApply: false
---

# Debug 66 - JavaScript Implementation

## Logging Setup

```javascript
// D66 Logger utility
const D66 = {
    indent: 0,
    
    log(msg, ...args) {
        const prefix = '[D66]' + '  '.repeat(this.indent);
        console.log(prefix, msg, ...args);
    },
    
    state(obj, name) {
        const type = Array.isArray(obj) ? 'Array' : typeof obj;
        const summary = this.summarize(obj);
        this.log(`DATA: ${name} | ${type} | ${summary}`);
    },
    
    summarize(obj, maxLen = 100) {
        if (obj === null) return 'null';
        if (obj === undefined) return 'undefined';
        if (Array.isArray(obj)) return `Array[${obj.length}]`;
        if (typeof obj === 'object') {
            const keys = Object.keys(obj);
            return `{${keys.length} keys: ${keys.slice(0, 3).join(', ')}${keys.length > 3 ? '...' : ''}}`;
        }
        if (typeof obj === 'string') return obj.length > 50 ? `"${obj.slice(0, 50)}..."` : `"${obj}"`;
        return String(obj).slice(0, maxLen);
    },
    
    enter(name) {
        this.log(`─── ENTER ${name} ───────────────────`);
        this.indent++;
        return performance.now();
    },
    
    exit(name, startTime, result) {
        this.indent = Math.max(0, this.indent - 1);
        const duration = ((performance.now() - startTime) / 1000).toFixed(3);
        this.log(`─── EXIT ${name} (${duration}s) → ${this.summarize(result)} ───`);
    }
};
```

## Instrumentation Patterns

### Function Wrapper (Recommended)

```javascript
function d66Trace(fn, name = fn.name) {
    return function(...args) {
        // [D66:START]
        const startTime = D66.enter(name);
        args.forEach((arg, i) => D66.log(`ARG[${i}]:`, D66.summarize(arg)));
        // [D66:END]
        
        try {
            const result = fn.apply(this, args);
            
            // [D66:START]
            D66.exit(name, startTime, result);
            // [D66:END]
            
            return result;
        } catch (e) {
            // [D66:START]
            D66.log(`ERROR in ${name}: ${e.message}`);
            D66.indent = Math.max(0, D66.indent - 1);
            // [D66:END]
            throw e;
        }
    };
}
```

### Async Function Wrapper

```javascript
function d66TraceAsync(fn, name = fn.name) {
    return async function(...args) {
        // [D66:START]
        const startTime = D66.enter(`(async) ${name}`);
        args.forEach((arg, i) => D66.log(`ARG[${i}]:`, D66.summarize(arg)));
        // [D66:END]
        
        try {
            const result = await fn.apply(this, args);
            
            // [D66:START]
            D66.exit(`(async) ${name}`, startTime, result);
            // [D66:END]
            
            return result;
        } catch (e) {
            // [D66:START]
            D66.log(`ERROR in ${name}: ${e.message}`);
            D66.indent = Math.max(0, D66.indent - 1);
            // [D66:END]
            throw e;
        }
    };
}
```

### Manual Function Entry/Exit

```javascript
function myFunction(arg1, arg2) {
    // [D66:START] ─────────────────────────
    const _d66Start = performance.now();
    D66.log('─── ENTER myFunction ───────────────────');
    D66.log(`  ARG: arg1 = ${D66.summarize(arg1)}`);
    D66.log(`  ARG: arg2 = ${D66.summarize(arg2)}`);
    D66.indent++;
    // [D66:END] ───────────────────────────
    
    // ... function body ...
    const result = doWork();
    
    // [D66:START]
    D66.indent--;
    D66.log(`─── EXIT myFunction (${((performance.now() - _d66Start) / 1000).toFixed(3)}s) → ${D66.summarize(result)} ───`);
    // [D66:END]
    
    return result;
}
```

### Object/Array State Inspection

```javascript
function d66ObjectState(obj, name) {
    // [D66:START]
    D66.log(`DATA: ${name}`);
    D66.log(`  type: ${typeof obj}`);
    if (Array.isArray(obj)) {
        D66.log(`  length: ${obj.length}`);
        if (obj.length > 0) {
            D66.log(`  first: ${JSON.stringify(obj[0])}`);
            if (obj.length > 1) D66.log(`  last: ${JSON.stringify(obj[obj.length - 1])}`);
        }
    } else if (obj && typeof obj === 'object') {
        const keys = Object.keys(obj);
        D66.log(`  keys: [${keys.join(', ')}]`);
        D66.log(`  preview: ${JSON.stringify(obj).slice(0, 200)}`);
    }
    // [D66:END]
}
```

### Loop Instrumentation

```javascript
// [D66:START]
const _d66Total = items.length;
const _d66Interval = Math.max(1, Math.floor(_d66Total / 10)); // Log every 10%
// [D66:END]

for (let i = 0; i < items.length; i++) {
    // [D66:START]
    if (i === 0 || i === _d66Total - 1 || i % _d66Interval === 0) {
        D66.log(`ITER: [${i + 1}/${_d66Total}] processing: ${D66.summarize(items[i])}`);
    }
    // [D66:END]
    
    const result = process(items[i]);
    
    // [D66:START]
    if (i === 0) {
        D66.log(`ITER: first result = ${D66.summarize(result)}`);
    }
    // [D66:END]
}
```

### Promise Chain Debugging

```javascript
fetchData()
    // [D66:START]
    .then(data => { D66.log('STEP: fetchData resolved', D66.summarize(data)); return data; })
    // [D66:END]
    .then(transformData)
    // [D66:START]
    .then(data => { D66.log('STEP: transformData resolved', D66.summarize(data)); return data; })
    // [D66:END]
    .then(saveData)
    // [D66:START]
    .catch(e => { D66.log(`ERROR in promise chain: ${e.message}`); throw e; });
    // [D66:END]
```

### Error Handling (Hybrid)

```javascript
// [D66:START]
try {
    // [D66:END]
    
    // ... original risky code ...
    const result = riskyOperation(data);
    
    // [D66:START]
} catch (e) {
    D66.log(`ERROR: ${e.name}: ${e.message}`);
    D66.log(`ERROR STATE: data = ${D66.summarize(data)}`);
    D66.log(`ERROR STACK: ${e.stack?.split('\n').slice(0, 3).join(' <- ')}`);
    throw e; // Re-throw to preserve stack
}
// [D66:END]
```

### Class Method Instrumentation

```javascript
class MyClass {
    process(data) {
        // [D66:START] ─────────────────────────
        const _d66Start = performance.now();
        D66.log(`─── ENTER ${this.constructor.name}.process ───`);
        D66.log(`  ARG: data = ${D66.summarize(data)}`);
        D66.log(`  STATE: this.config = ${JSON.stringify(this.config)}`);
        D66.indent++;
        // [D66:END] ───────────────────────────
        
        // ... method body ...
        
        // [D66:START]
        D66.indent--;
        D66.log(`─── EXIT ${this.constructor.name}.process (${((performance.now() - _d66Start) / 1000).toFixed(3)}s) ───`);
        // [D66:END]
    }
}
```

---

## Bitburner Extensions

Bitburner uses a custom NS (Netscript) API. These patterns are designed for debugging scripts in the game.

### NS-Compatible Logger

```javascript
/** @param {NS} ns */
export async function main(ns) {
    // Bitburner-compatible D66 logger
    const D66 = {
        indent: 0,
        log(msg) {
            const prefix = '[D66]' + '  '.repeat(this.indent);
            ns.tprint(prefix + ' ' + msg);  // Use ns.tprint for terminal output
            // Or ns.print() for script log only
        },
        enter(name) { this.log(`─── ENTER ${name} ───`); this.indent++; return Date.now(); },
        exit(name, start) { this.indent--; this.log(`─── EXIT ${name} (${Date.now() - start}ms) ───`); }
    };
    
    // Your script here...
}
```

### RAM Cost Tracking

```javascript
/** @param {NS} ns */
export async function main(ns) {
    // [D66:START]
    const scriptName = ns.getScriptName();
    const scriptRAM = ns.getScriptRam(scriptName);
    D66.log(`─── SCRIPT: ${scriptName} ───`);
    D66.log(`  RAM cost: ${scriptRAM.toFixed(2)} GB`);
    D66.log(`  Host: ${ns.getHostname()}`);
    D66.log(`  Available RAM: ${ns.getServerMaxRam(ns.getHostname()) - ns.getServerUsedRam(ns.getHostname())} GB`);
    // [D66:END]
}
```

### Server Analysis

```javascript
/** @param {NS} ns */
function d66ServerState(ns, hostname) {
    // [D66:START]
    D66.log(`SERVER: ${hostname}`);
    D66.log(`  RAM: ${ns.getServerUsedRam(hostname).toFixed(1)}/${ns.getServerMaxRam(hostname)} GB`);
    D66.log(`  Security: ${ns.getServerSecurityLevel(hostname).toFixed(2)} (min: ${ns.getServerMinSecurityLevel(hostname)})`);
    D66.log(`  Money: $${ns.formatNumber(ns.getServerMoneyAvailable(hostname))} / $${ns.formatNumber(ns.getServerMaxMoney(hostname))}`);
    D66.log(`  Hack chance: ${(ns.hackAnalyzeChance(hostname) * 100).toFixed(1)}%`);
    D66.log(`  Root access: ${ns.hasRootAccess(hostname)}`);
    // [D66:END]
}
```

### Hack/Grow/Weaken Cycle Debugging

```javascript
/** @param {NS} ns */
export async function main(ns) {
    const target = ns.args[0];
    
    while (true) {
        // [D66:START]
        D66.log(`─── CYCLE START: ${target} ───`);
        d66ServerState(ns, target);
        // [D66:END]
        
        const securityThresh = ns.getServerMinSecurityLevel(target) + 5;
        const moneyThresh = ns.getServerMaxMoney(target) * 0.75;
        
        if (ns.getServerSecurityLevel(target) > securityThresh) {
            // [D66:START]
            D66.log(`  ACTION: weaken (security ${ns.getServerSecurityLevel(target).toFixed(2)} > ${securityThresh})`);
            const weakenTime = ns.getWeakenTime(target);
            D66.log(`  WAIT: ${(weakenTime / 1000).toFixed(1)}s`);
            // [D66:END]
            
            await ns.weaken(target);
            
            // [D66:START]
            D66.log(`  RESULT: security now ${ns.getServerSecurityLevel(target).toFixed(2)}`);
            // [D66:END]
        } else if (ns.getServerMoneyAvailable(target) < moneyThresh) {
            // [D66:START]
            D66.log(`  ACTION: grow (money $${ns.formatNumber(ns.getServerMoneyAvailable(target))} < $${ns.formatNumber(moneyThresh)})`);
            // [D66:END]
            
            await ns.grow(target);
            
            // [D66:START]
            D66.log(`  RESULT: money now $${ns.formatNumber(ns.getServerMoneyAvailable(target))}`);
            // [D66:END]
        } else {
            // [D66:START]
            const hackAmount = ns.hackAnalyze(target) * ns.getServerMoneyAvailable(target);
            D66.log(`  ACTION: hack (expected: $${ns.formatNumber(hackAmount)})`);
            // [D66:END]
            
            const stolen = await ns.hack(target);
            
            // [D66:START]
            D66.log(`  RESULT: stole $${ns.formatNumber(stolen)}`);
            // [D66:END]
        }
        
        // [D66:START]
        D66.log(`─── CYCLE END ───\n`);
        // [D66:END]
    }
}
```

### Port Communication Debugging

```javascript
/** @param {NS} ns */
function d66PortState(ns, portNum) {
    // [D66:START]
    const handle = ns.getPortHandle(portNum);
    D66.log(`PORT ${portNum}:`);
    D66.log(`  empty: ${handle.empty()}`);
    D66.log(`  full: ${handle.full()}`);
    D66.log(`  peek: ${handle.empty() ? 'N/A' : JSON.stringify(handle.peek())}`);
    // [D66:END]
}

// Writing to port
// [D66:START]
D66.log(`PORT WRITE: port ${portNum}`);
D66.log(`  data: ${JSON.stringify(data)}`);
// [D66:END]
const success = ns.tryWritePort(portNum, data);
// [D66:START]
D66.log(`  success: ${success}`);
// [D66:END]
```

### Batch Script Coordination

```javascript
/** @param {NS} ns */
export async function main(ns) {
    const [target, type, delay, batchId] = ns.args;
    
    // [D66:START]
    D66.log(`─── BATCH ${batchId} | ${type.toUpperCase()} ───`);
    D66.log(`  target: ${target}`);
    D66.log(`  delay: ${delay}ms`);
    D66.log(`  scheduled: ${new Date().toISOString()}`);
    // [D66:END]
    
    await ns.sleep(delay);
    
    // [D66:START]
    D66.log(`  executing at: ${new Date().toISOString()}`);
    const _d66Start = Date.now();
    // [D66:END]
    
    let result;
    switch (type) {
        case 'hack': result = await ns.hack(target); break;
        case 'grow': result = await ns.grow(target); break;
        case 'weaken': result = await ns.weaken(target); break;
    }
    
    // [D66:START]
    D66.log(`  completed in: ${Date.now() - _d66Start}ms`);
    D66.log(`  result: ${result}`);
    D66.log(`─── BATCH ${batchId} DONE ───`);
    // [D66:END]
}
```

### Script Spawning Debug

```javascript
// [D66:START]
D66.log(`─── SPAWN ───`);
D66.log(`  script: ${scriptPath}`);
D66.log(`  host: ${host}`);
D66.log(`  threads: ${threads}`);
D66.log(`  args: ${JSON.stringify(args)}`);
const ramNeeded = ns.getScriptRam(scriptPath) * threads;
const ramAvail = ns.getServerMaxRam(host) - ns.getServerUsedRam(host);
D66.log(`  RAM needed: ${ramNeeded.toFixed(2)} GB`);
D66.log(`  RAM available: ${ramAvail.toFixed(2)} GB`);
D66.log(`  can run: ${ramAvail >= ramNeeded}`);
// [D66:END]

const pid = ns.exec(scriptPath, host, threads, ...args);

// [D66:START]
D66.log(`  PID: ${pid > 0 ? pid : 'FAILED'}`);
// [D66:END]
```

---

## Cleanup

Remove all Debug 66 instrumentation:

```bash
# Remove D66 lines
grep -v "D66" file.js > file_clean.js

# Remove D66 block comments
sed '/\/\/ \[D66:START\]/,/\/\/ \[D66:END\]/d' file.js > file_clean.js
```

## Quick Copy-Paste Templates

### Minimal Function Wrapper
```javascript
// [D66:START]
D66.log(`─── ENTER %FNAME% ───`); const _d66t = performance.now(); D66.indent++;
// [D66:END]
// ... body ...
// [D66:START]
D66.indent--; D66.log(`─── EXIT %FNAME% (${((performance.now() - _d66t)/1000).toFixed(3)}s) ───`);
// [D66:END]
```

### Bitburner Minimal
```javascript
// [D66:START]
ns.tprint(`[D66] ─── ENTER %FNAME% ───`);
// [D66:END]
```
