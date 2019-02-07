'use strict'

const cps = require('../index')
const sampleProofFromFile = require('../docs/samples/chainpoint-proof-v3.chp.json')

let sampleProof

beforeEach(function () {
  // Clone the sample so properties can be modified/deleted
  // without touching the original.
  sampleProof = JSON.parse(JSON.stringify(sampleProofFromFile))
})

describe('sample proof', function () {
  it('should be valid', function (done) {
    cps.validate(sampleProof).should.have.property('valid', true)
    cps.validate(sampleProof).should.have.property('errors', null)
    done()
  })
})

describe('proof root', function () {
  it('should be invalid with missing @context', function (done) {
    delete sampleProof['@context']
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data["@context"]')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'is required')
    done()
  })

  it('should be invalid with unknown @context', function (done) {
    sampleProof['@context'] = 'foo'
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data["@context"]')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'must be an enum value')
    done()
  })

  it('should be invalid with missing type', function (done) {
    delete sampleProof['type']
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.type')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'is required')
    done()
  })

  it('should be invalid with unknown type', function (done) {
    sampleProof['type'] = 'foo'
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.type')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'must be an enum value')
    done()
  })

  it('should be invalid with missing hash', function (done) {
    delete sampleProof['hash']
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'is required')
    done()
  })

  it('should be invalid with non-hex hash', function (done) {
    sampleProof['hash'] = 'xyz'
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'pattern mismatch')
    done()
  })

  it('should be invalid with missing hash_id_node', function (done) {
    delete sampleProof['hash_id_node']
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash_id_node')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'is required')
    done()
  })

  it('should be invalid with non-UUID hash_id_node', function (done) {
    sampleProof['hash_id_node'] = 'abc'
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash_id_node')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'pattern mismatch')
    done()
  })

  it('should be invalid with missing hash_submitted_node_at', function (done) {
    delete sampleProof['hash_submitted_node_at']
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash_submitted_node_at')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'is required')
    done()
  })

  it('should be invalid with non ISO8601 date hash_submitted_node_at', function (done) {
    sampleProof['hash_submitted_node_at'] = 'March 1, 2017'
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash_submitted_node_at')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'pattern mismatch')
    done()
  })

  it('should be invalid with ISO8601 date hash_submitted_node_at in non-strict millisecond granularity form', function (done) {
    sampleProof['hash_submitted_node_at'] = '2017-04-25T19:10:07.171Z'
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash_submitted_node_at')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'pattern mismatch')
    done()
  })

  it('should be invalid with missing hash_id_core', function (done) {
    delete sampleProof['hash_id_core']
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash_id_core')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'is required')
    done()
  })

  it('should be invalid with non-UUID hash_id_core', function (done) {
    sampleProof['hash_id_core'] = 'abc'
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash_id_core')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'pattern mismatch')
    done()
  })

  it('should be invalid with missing hash_submitted_core_at', function (done) {
    delete sampleProof['hash_submitted_core_at']
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash_submitted_core_at')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'is required')
    done()
  })

  it('should be invalid with non ISO8601 date hash_submitted_core_at', function (done) {
    sampleProof['hash_submitted_core_at'] = 'March 1, 2017'
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash_submitted_core_at')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'pattern mismatch')
    done()
  })

  it('should be invalid with ISO8601 date hash_submitted_core_at in non-strict millisecond granularity form', function (done) {
    sampleProof['hash_submitted_core_at'] = '2017-04-25T19:10:07.171Z'
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.hash_submitted_core_at')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'pattern mismatch')
    done()
  })

  it('should be invalid with missing branches at the root', function (done) {
    delete sampleProof['branches']
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.branches')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'is required')
    done()
  })

  it('should be invalid with an added property at the root', function (done) {
    sampleProof.extra = {}
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'has additional properties')
    done()
  })
})

describe('proof.branches', function () {
  it('should be valid with optional label removed', function (done) {
    let branch = sampleProof.branches[0]
    delete branch.label
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with optional branches removed', function (done) {
    let branch = sampleProof.branches[0]
    delete branch.branches
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be invalid with required ops removed', function (done) {
    let branch = sampleProof.branches[0]
    delete branch.ops
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.branches.0')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'referenced schema does not match')
    done()
  })

  it('should be invalid with an additional property added', function (done) {
    let branch = sampleProof.branches[0]
    branch.extra = {}
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.branches.0')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'referenced schema does not match')
    done()
  })
})

describe('proof.branches[0].ops', function () {
  it('should be valid if empty', function (done) {
    sampleProof.branches[0].ops = []
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a single approved l property', function (done) {
    sampleProof.branches[0].ops = [{l: 'abc'}]
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a single approved r property', function (done) {
    sampleProof.branches[0].ops = [{r: 'abc'}]
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a single approved op property with an sha-224 hash', function (done) {
    sampleProof.branches[0].ops = [{op: 'sha-224'}]
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a single approved op property with an sha-256 hash', function (done) {
    sampleProof.branches[0].ops = [{op: 'sha-256'}]
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a single approved op property with an sha-384 hash', function (done) {
    sampleProof.branches[0].ops = [{op: 'sha-384'}]
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a single approved op property with an sha-512 hash', function (done) {
    sampleProof.branches[0].ops = [{op: 'sha-512'}]
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a single approved op property with an sha3-224 hash', function (done) {
    sampleProof.branches[0].ops = [{op: 'sha-224'}]
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a single approved op property with an sha3-256 hash', function (done) {
    sampleProof.branches[0].ops = [{op: 'sha-256'}]
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a single approved op property with an sha3-384 hash', function (done) {
    sampleProof.branches[0].ops = [{op: 'sha-384'}]
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a single approved op property with an sha3-512 hash', function (done) {
    sampleProof.branches[0].ops = [{op: 'sha-512'}]
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a single approved op property with an sha-256-x2 hash', function (done) {
    sampleProof.branches[0].ops = [{op: 'sha-512'}]
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be invalid with a single approved op property with an unknown hash', function (done) {
    sampleProof.branches[0].ops = [{op: 'blake2s'}]
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.branches.0')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'referenced schema does not match')
    done()
  })

  it('should be invalid with an additional property added', function (done) {
    sampleProof.branches[0].ops = [{foo: 'bar'}]
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.branches.0')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'referenced schema does not match')
    done()
  })
})

describe('proof.branches[0].ops[0].anchors', function () {
  it('should be valid if empty', function (done) {
    sampleProof.branches[0].ops[0].anchors = []
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a cal type', function (done) {
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].type = 'cal'
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a eth type', function (done) {
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].type = 'eth'
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with a btc type', function (done) {
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].type = 'btc'
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be invalid with an empty type', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].type = null
    cps.validate(sampleProof).should.have.property('valid', false)
    done()
  })

  it('should be invalid with an incorrectly cased type', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].type = 'bTc'
    cps.validate(sampleProof).should.have.property('valid', false)
    done()
  })

  it('should be invalid with a too short type', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].type = 'ab'
    cps.validate(sampleProof).should.have.property('valid', false)
    done()
  })

  it('should be invalid with a too long type', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].type = 'abcdefghijk'
    cps.validate(sampleProof).should.have.property('valid', false)
    done()
  })

  it('should be valid with an arbitrary anchor type', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].type = 'foo'
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with an unknown anchor_id', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].anchor_id = 'foo'
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be invalid with an empty anchor_id', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].anchor_id = null
    cps.validate(sampleProof).should.have.property('valid', false)
    done()
  })

  it('should be invalid with an integer anchor_id', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].anchor_id = 123
    cps.validate(sampleProof).should.have.property('valid', false)
    done()
  })

  it('should be valid with no optional uris', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    let anchor = sampleProof.branches[0].ops[lastAnchorIndex].anchors[0]
    delete anchor.uris
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with empty optional uris', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].uris = []
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be valid with valid optional uris', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].uris = ['https://a.cal.chainpoint.org', 'https://b.cal.chainpoint.org']
    cps.validate(sampleProof).should.have.property('valid', true)
    done()
  })

  it('should be invalid with malformed optional uris', function (done) {
    // get the last ops, which should be an anchor
    let lastAnchorIndex = sampleProof.branches[0].ops.length - 1
    sampleProof.branches[0].ops[lastAnchorIndex].anchors[0].uris = ['foo', 'bar']
    cps.validate(sampleProof).should.have.property('valid', false)
    cps.validate(sampleProof).errors[0].should.have.property('field', 'data.branches.0')
    cps.validate(sampleProof).errors[0].should.have.property('message', 'referenced schema does not match')
    done()
  })
})
