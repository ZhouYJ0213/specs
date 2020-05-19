createJsonDownloadButton(combos, 'nse.json', 'Download Ubercalc query')

createJsonDownloadButton(solved_many, 'nse-results.json', "Download Ubercalc results")

report_from_result(vars, solved_many[0], {})

viewof filter = html`<input type="text">`

orientableDebugger(await fetch_model(), solved_many, vars, filter)

md`### Ubercalc`

solved_many_pre = (await solve_many_chunk(combos))

solved_many = solved_many_pre.filter(d => d !== null)

combos = makeQuery([
  {
    "!NSE": true,
    "!SDR": false,

    windows: 256,
    window_size_gib: 4,
    nodes_in_sequence: 8,

    post_window_challenges: 2,
    porep_lambda: 10,
    spacegap: 0.15,
    delta: 0.05,

    expander_degree: 384,
    butterfly_degree: 16,
    expander_layers: 8,
    butterfly_layers: 7,

    rig_cost: 4750,
  },
  {
    "!SDR": true,
    "!NSE": false,
    replica_size_gib: 32,
    porep_partitions: 8,
    wpost_sectors: 2350,
    porep_time_commitment: (150 * 8)/4, // 150 for 4GiB, 4 GPU parallelization

    cost_attack_replica: 0.015 / 10,
    layers: 11,
    parents: 37,

    rig_cost: 2000,
  },
])
  .add({
    rig_cost_gpu: 3000,
    rig_memaccess_throughput_tb_s:  3,
    rig_hashing_throughput_tb_s: 0.016 * 32,
    rig_lifetime_years: 2,
    rig_storage_lifetime_years: 2,
    rig_cost_storage_tb: 15,
    rig_hashing_sequential_throughput_gb_s: 2.5,

    mtree_hash_name: 'poseidon',
    mtree_hash_time: 8.3e-7 / 4, // (divided by 4 since we have 4 GPUs) // ((8/7)*(2^27/8 -1))*32*8, // GPU 4s per GiB // CPU 5.803e-5,
    mtree_hash_blocks: 8,
    mtree_hash_constraints: 508 + 56,
    kdf_constraints: 25849/2,
    commd_hash_name: 'sha',
    commd_hash_constraints: 25840,
    commd_hash_time: 130e-9,

    node_size: 32,
    snark_partition: 100000000,
    snark_constraint_time: 0.00000317488 / 4,
    snark_size: 192,

    proving_period_hours: 24,
    network_size_eib: 10,
    block_time: 30,
    tipset_size: 1,
  })
  .compile()

function fetch_model() {
  return fetch('https://raw.githubusercontent.com/filecoin-project/specs/nse-calc/src/orient/nse.orient?time=' + new Date()).then(d => d.text())
}

md`### Orient`

vars = dump_vars()

orientServer = `http://${window.location.hostname}:8000`

md`### Imports`

import { createJsonDownloadButton } from "@trebor/download-json"

html`<style>
.markdown-body table td, .markdown-body table t {
  padding: 4px !important;
}
table {
  font-size: 12px
}
th {
  font-size: 10px;
}
label {
  font-size: 10px;
}
</style>`

import {makeQuery, solve_many_chunk, report_from_result, dump_vars} from "@nicola/orientable-utils"

import {orientableDebugger} from "@nicola/orientable-debugger"