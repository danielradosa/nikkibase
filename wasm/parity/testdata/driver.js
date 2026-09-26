"use strict";

const fs = require("fs");

globalThis.require = require;
globalThis.fs = fs;
globalThis.path = require("path");
globalThis.TextEncoder = require("util").TextEncoder;
globalThis.TextDecoder = require("util").TextDecoder;
globalThis.performance ??= require("perf_hooks").performance;
globalThis.crypto ??= require("crypto");

const [shim, modulePath, inputPath, outputPath] = process.argv.slice(2);
require(shim);

const input = JSON.parse(fs.readFileSync(inputPath, "utf8"));
const bytes = (base64) => new Uint8Array(Buffer.from(base64, "base64"));

function unwrap(res) {
  if (!res.ok) throw new Error(res.error);
  return JSON.parse(res.json);
}

const refused = (res) => (res.ok ? "accepted" : res.error);

function worth(w) {
  const ids = unwrap(nikkibase.importSelections(w.wardrobe)).ids;
  const runs = w.runs.map((settings) => {
    const start = unwrap(nikkibase.worthStart(w.versions, settings, w.suits));
    const progress = [];
    for (;;) {
      const step = unwrap(nikkibase.worthRun(start.session, w.chunk));
      progress.push(step);
      if (step.done === step.total) break;
    }
    return {
      total: start.total,
      session: start.session,
      progress,
      ranks: w.filters.map((filter) => unwrap(nikkibase.worthRank(start.session, filter, w.limit))),
    };
  });
  const live = unwrap(nikkibase.worthStart(w.versions, null)).session;
  const rejected = [
    ...w.bad.start.map(([versions, settings, suits]) => refused(nikkibase.worthStart(versions, settings, suits))),
    ...w.bad.run.map(([offset, count]) => refused(nikkibase.worthRun(live + offset, count))),
    ...w.bad.rank.map(([offset, filter, limit]) => refused(nikkibase.worthRank(live + offset, filter, limit))),
    refused(nikkibase.worthStart(5n, null)),
    refused(nikkibase.worthStart([{ ...w.versions[0], weights: [1n, 1, 1, 1, 1] }], null)),
    refused(nikkibase.worthStart([{ ...w.versions[0], key: 3n }], null)),
    refused(nikkibase.worthStart(w.versions, 5n)),
    refused(nikkibase.worthStart(w.versions, { auto: true, levels: { smile: 3n } })),
    refused(nikkibase.worthRun(BigInt(live), 2)),
    refused(nikkibase.worthRank(live, 3n, 5)),
    refused(nikkibase.worthRank(live, { modes: [1n] }, 5)),
    refused(nikkibase.worthRank(live, { places: [1n] }, 5)),
    refused(nikkibase.worthRank(live, {}, 5n)),
    refused(nikkibase.worthStart(w.versions, null, 5n)),
    refused(nikkibase.worthStart(w.versions, null, [{ key: 1n, items: [] }])),
    refused(nikkibase.worthStart(w.versions, null, [{ key: "a", items: [1n] }])),
    refused(nikkibase.worthRank(live, { suits: 1n }, 5)),
  ];
  const kept = refused(nikkibase.worthRank(live, {}, 1));
  unwrap(nikkibase.setWardrobe(ids));
  const dropped = [refused(nikkibase.worthRun(live, 1)), refused(nikkibase.worthRank(live, {}, 1))];
  return { runs, rejected, kept, dropped };
}

const go = new Go();
WebAssembly.instantiate(fs.readFileSync(modulePath), go.importObject)
  .then((wasm) => {
    go.run(wasm.instance);

    const early = refused(nikkibase.places());
    unwrap(nikkibase.loadCatalogue(bytes(input.catalogue)));
    const result = {
      early,
      places: unwrap(nikkibase.places()),
      decoded: unwrap(nikkibase.importSelections(input.wardrobe)),
      owned: unwrap(nikkibase.best(input.weights, input.attrs, input.skills, input.tags)),
      all: unwrap(nikkibase.best(input.weights, input.attrs, input.skills, input.tags, "all")),
      ideal: unwrap(nikkibase.best(input.weights, input.attrs, null, input.tags, "all")),
      placed: input.placed.map((skills) => ({
        owned: unwrap(nikkibase.best(input.weights, input.attrs, skills, input.tags)),
        all: unwrap(nikkibase.best(input.weights, input.attrs, skills, input.tags, "all")),
      })),
      rejected: input.bad.map((skills) => {
        const res = nikkibase.best(input.weights, input.attrs, skills, input.tags);
        return res.ok ? "accepted" : res.error;
      }),
      required: input.required.map((run) => ({
        owned: unwrap(nikkibase.best(input.weights, input.attrs, run.skills, input.tags, "wardrobe", run.require)),
        all: unwrap(nikkibase.best(input.weights, input.attrs, run.skills, input.tags, "all", run.require)),
      })),
    };
    result.worth = worth(input.worth);
    fs.writeFileSync(outputPath, JSON.stringify(result));
    process.exit(0);
  })
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });
