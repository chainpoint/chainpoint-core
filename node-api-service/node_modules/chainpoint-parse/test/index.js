/* global describe, it */

const should = require('should')
const chpParse = require('../index')
const fs = require('fs')
const chpBinary = require('chainpoint-binary')

describe('Using a valid chainpoint v3 JSON file, parse as Object', function () {
  it('should return expected results', function (done) {
    let jsonSample = fs.readFileSync('./test/data/chainpoint-proof-v3.chp.json', 'utf-8')
    should.exist(jsonSample)
    let result = chpParse.parse(JSON.parse(jsonSample))
    result.branches.length.should.equal(1)
    result.branches[0].anchors.length.should.equal(1)
    result.branches[0].branches.length.should.equal(1)
    result.branches[0].branches[0].anchors.length.should.equal(1)
    result.branches[0].branches[0].anchors[0].expected_value.should.equal('c617f5faca34474bea7020d75c39cb8427a32145f9646586ecb9184002131ad9')
    should.not.exist(result.branches[0].branches[0].branches)
    done()
  })
})

describe('Using a valid chainpoint v3 JSON file, parse as Buffer', function () {
  it('should return expected results', function (done) {
    let jsonSample = fs.readFileSync('./test/data/chainpoint-proof-v3.chp.json', 'utf-8')
    should.exist(jsonSample)
    let bufferProof = chpBinary.objectToBinarySync(JSON.parse(jsonSample))
    let result = chpParse.parse(bufferProof)
    result.branches.length.should.equal(1)
    result.branches[0].anchors.length.should.equal(1)
    result.branches[0].branches.length.should.equal(1)
    result.branches[0].branches[0].anchors.length.should.equal(1)
    result.branches[0].branches[0].anchors[0].expected_value.should.equal('c617f5faca34474bea7020d75c39cb8427a32145f9646586ecb9184002131ad9')
    should.not.exist(result.branches[0].branches[0].branches)
    done()
  })
})

describe('Using a valid chainpoint v3 JSON file, parse as Hex string', function () {
  it('should return expected results', function (done) {
    let jsonSample = fs.readFileSync('./test/data/chainpoint-proof-v3.chp.json', 'utf-8')
    should.exist(jsonSample)
    let bufferProof = chpBinary.objectToBinarySync(JSON.parse(jsonSample))
    let result = chpParse.parse(bufferProof.toString('hex'))
    result.branches.length.should.equal(1)
    result.branches[0].anchors.length.should.equal(1)
    result.branches[0].branches.length.should.equal(1)
    result.branches[0].branches[0].anchors.length.should.equal(1)
    result.branches[0].branches[0].anchors[0].expected_value.should.equal('c617f5faca34474bea7020d75c39cb8427a32145f9646586ecb9184002131ad9')
    should.not.exist(result.branches[0].branches[0].branches)
    done()
  })
})

describe('Using a valid chainpoint v3 binary Base 64 file, parse as Base64', function () {
  it('should return expected results', function (done) {
    let b64Sample = fs.readFileSync('./test/data/chainpoint-proof-v3.chp.b64', 'utf-8')
    should.exist(b64Sample)
    let result = chpParse.parse(b64Sample)
    result.branches.length.should.equal(1)
    result.branches[0].anchors.length.should.equal(1)
    result.branches[0].branches.length.should.equal(1)
    result.branches[0].branches[0].anchors.length.should.equal(1)
    result.branches[0].branches[0].anchors[0].expected_value.should.equal('c617f5faca34474bea7020d75c39cb8427a32145f9646586ecb9184002131ad9')
    should.not.exist(result.branches[0].branches[0].branches)
    done()
  })
})
