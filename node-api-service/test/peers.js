/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const peers = require('../lib/endpoints/peers.js')

describe('Peers Controller', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync()
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('GET /peers with bad TM connection', () => {
    before(() => {
      peers.setTmRpc({
        getNetInfoAsync: async () => {
          return { error: true }
        }
      })
    })
    it('should return proper error with TM communication error', done => {
      request(insecureServer)
        .get('/peers')
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
            .and.to.equal('Could not query for net info')
          done()
        })
    })
  })

  describe('GET /peers', () => {
    let peerData = [
      { remote_ip: '65.125.23.1', node_info: { listen_addr: '65.125.23.1' } },
      { remote_ip: '65.125.23.2', node_info: { listen_addr: '65.125.23.2' } },
      { remote_ip: '65.125.23.3', node_info: { listen_addr: '65.125.23.3' } }
    ]
    let expectedResults = ['65.125.23.1', '65.125.23.2', '65.125.23.3']
    before(() => {
      peers.setTmRpc({
        getNetInfoAsync: async () => {
          return { result: { peers: peerData } }
        }
      })
    })
    it('should return proper peers array', done => {
      request(insecureServer)
        .get('/peers')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.be.a('array')
            .and.to.deep.equal(expectedResults)
          done()
        })
    })
  })

  describe('GET /peers with filtering', () => {
    let peerData = [
      { remote_ip: '65.125.23.1', node_info: { listen_addr: '65.125.23.1' } },
      { remote_ip: '65.125.23.2', node_info: { listen_addr: '' } },
      { remote_ip: '65.125.23.3', node_info: { listen_addr: '' } },
      { remote_ip: '10.125.23.4', node_info: { listen_addr: '65.125.23.4:26656' } },
      { remote_ip: '192.125.23.5', node_info: { listen_addr: 'http://65.125.23.5:26656' } },
      { remote_ip: '192.125.23.6', node_info: { listen_addr: '192.125.23.6:26656' } }
    ]
    let expectedResults = ['65.125.23.1', '65.125.23.2', '65.125.23.3', '65.125.23.4', '65.125.23.5']
    before(() => {
      peers.setTmRpc({
        getNetInfoAsync: async () => {
          return { result: { peers: peerData } }
        }
      })
    })
    it('should return proper filtered peers array', done => {
      request(insecureServer)
        .get('/peers')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.be.a('array')
            .and.to.deep.equal(expectedResults)
          done()
        })
    })
  })
})
