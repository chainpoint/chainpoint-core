/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const status = require('../lib/endpoints/status.js')
const { version } = require('../package.json')

describe('Status Controller - Public Mode', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(false)
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('GET /status with bad TM connection', () => {
    before(() => {
      status.setTmRpc({
        getStatusAsync: async () => {
          return { error: true }
        }
      })
    })
    it('should return proper error with TM communication error', done => {
      request(insecureServer)
        .get('/status')
        .expect('Content-type', /json/)
        .expect(500)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InternalServer')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('Could not query for status')
          done()
        })
    })
  })

  describe('GET /status', () => {
    let baseURI = 'http://base.uri'
    before(() => {
      status.setENV({ CHAINPOINT_CORE_BASE_URI: baseURI, NETWORK: 'testnet', PRIVATE_NETWORK: false })
      let statusResult = { tmresult: 1 }
      status.setTmRpc({
        getStatusAsync: async () => {
          return { result: statusResult }
        }
      })
    })
    it('should return proper status object', done => {
      request(insecureServer)
        .get('/status')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('version')
            .and.to.be.a('string')
            .and.to.equal(version)
          expect(res.body)
            .to.have.property('time')
            .and.to.be.a('string')
          expect(res.body)
            .to.have.property('base_uri')
            .and.to.be.a('string')
            .and.to.equal(baseURI)
          expect(res.body)
            .to.have.property('network')
            .and.to.be.a('string')
            .and.to.equal('testnet')
          expect(res.body)
            .to.have.property('mode')
            .and.to.be.a('string')
            .and.to.equal('public')
          expect(res.body)
            .to.have.property('tmresult')
            .and.to.be.a('number')
            .and.to.equal(1)
          done()
        })
    })
  })
})

describe('Status Controller - Private Mode', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(true)
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('GET /status', () => {
    let baseURI = 'http://base.uri'
    before(() => {
      status.setENV({ CHAINPOINT_CORE_BASE_URI: baseURI, NETWORK: 'testnet', PRIVATE_NETWORK: true })
      let statusResult = { tmresult: 1 }
      status.setTmRpc({
        getStatusAsync: async () => {
          return { result: statusResult }
        }
      })
    })
    it('should return proper status object', done => {
      request(insecureServer)
        .get('/status')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('version')
            .and.to.be.a('string')
            .and.to.equal(version)
          expect(res.body)
            .to.have.property('time')
            .and.to.be.a('string')
          expect(res.body)
            .to.have.property('base_uri')
            .and.to.be.a('string')
            .and.to.equal(baseURI)
          expect(res.body)
            .to.have.property('network')
            .and.to.be.a('string')
            .and.to.equal('testnet')
          expect(res.body)
            .to.have.property('mode')
            .and.to.be.a('string')
            .and.to.equal('private')
          expect(res.body)
            .to.have.property('tmresult')
            .and.to.be.a('number')
            .and.to.equal(1)
          done()
        })
    })
  })
})
