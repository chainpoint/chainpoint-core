/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const status = require('../lib/endpoints/status.js')
const { version } = require('../package.json')

describe('Status Controller', () => {
  let apiServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    apiServer = await app.startAPIServerAsync(false)
  })
  afterEach(() => {
    apiServer.close()
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
      request(apiServer)
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

  describe('GET /status with bad LND connection', () => {
    let baseURI = 'http://base.uri'
    before(() => {
      status.setENV({ CHAINPOINT_CORE_BASE_URI: baseURI, NETWORK: 'testnet' })
      let statusResult = { tmresult: 1 }
      status.setTmRpc({
        getStatusAsync: async () => {
          return { result: statusResult }
        }
      })
      status.setLND({
        services: {
          Lightning: {
            getInfo: () => {
              throw new Error('err!')
            }
          }
        }
      })
    })
    it('should return proper error with LND communication error', done => {
      request(apiServer)
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
      status.setENV({ CHAINPOINT_CORE_BASE_URI: baseURI, NETWORK: 'testnet' })
      let statusResult = { tmresult: 1 }
      status.setTmRpc({
        getStatusAsync: async () => {
          return { result: statusResult }
        }
      })
      status.setLND({
        services: {
          Lightning: {
            getInfo: () => {
              return {
                identity_pubkey: 'identity_pubkey',
                uris: ['uris'],
                num_active_channels: 1,
                alias: 'alias'
              }
            }
          }
        }
      })
    })
    it('should return proper status object', done => {
      request(apiServer)
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
            .to.have.property('tmresult')
            .and.to.be.a('number')
            .and.to.equal(1)
          expect(res.body)
            .to.have.property('identity_pubkey')
            .and.to.be.a('string')
            .and.to.equal('identity_pubkey')
          expect(res.body)
            .to.have.property('uris')
            .and.to.be.a('array')
          expect(res.body.uris).to.have.length(1)
          expect(res.body.uris[0])
            .and.to.be.a('string')
            .and.to.equal('uris')
          expect(res.body)
            .to.have.property('num_active_channels')
            .and.to.be.a('number')
            .and.to.equal(1)
          expect(res.body)
            .to.have.property('alias')
            .and.to.be.a('string')
            .and.to.equal('alias')
          done()
        })
    })
  })
})
