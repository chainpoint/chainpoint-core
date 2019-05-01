/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const nodes = require('../lib/endpoints/nodes.js')

describe('Nodes Controller', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync()
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('GET /nodes with bad TM connection for ABCI info', () => {
    let randomIps = ['65.10.123.1']
    let dbResult = randomIps.map(ip => {
      return { publicIp: ip }
    })
    let result = randomIps.map(ip => {
      return { public_uri: `http://${ip}` }
    })
    before(() => {
      nodes.setTmRpc({
        getAbciInfo: async () => {
          return { error: true }
        }
      })
      nodes.setStakedNode({ getRandomNodes: async () => dbResult })
    })
    it('should return random nodes', done => {
      request(insecureServer)
        .get('/nodes/random')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.be.a('array')
            .and.to.deep.equal(result)
          done()
        })
    })
  })

  describe('GET /nodes with bad TM connection for ABCI info', () => {
    let randomIps = ['65.10.123.1']
    let dbResult = randomIps.map(ip => {
      return { publicIp: ip }
    })
    let result = randomIps.map(ip => {
      return { public_uri: `http://${ip}` }
    })
    before(() => {
      nodes.setTmRpc({
        getAbciInfo: async () => {
          return { result: { response: { data: '{"last_mint_block":27,"prev_mint_block":27}' } } }
        },
        getTxSearch: async () => {
          return { error: true }
        }
      })
      nodes.setStakedNode({ getRandomNodes: async () => dbResult })
    })
    it('should return random nodes', done => {
      request(insecureServer)
        .get('/nodes/random')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.be.a('array')
            .and.to.deep.equal(result)
          done()
        })
    })
  })

  describe('GET /nodes with empty results', () => {
    let randomIps = ['65.10.123.1']
    let dbResult = randomIps.map(ip => {
      return { publicIp: ip }
    })
    let result = randomIps.map(ip => {
      return { public_uri: `http://${ip}` }
    })
    before(() => {
      nodes.setTmRpc({
        getAbciInfo: async () => {
          return { result: { response: { data: '{"last_mint_block":27,"prev_mint_block":27}' } } }
        },
        getTxSearch: async () => {
          return { result: { txs: [] } }
        }
      })
      nodes.setStakedNode({ getRandomNodes: async () => dbResult })
    })
    it('should return random nodes', done => {
      request(insecureServer)
        .get('/nodes/random')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.be.a('array')
            .and.to.deep.equal(result)
          done()
        })
    })
  })

  describe('GET /nodes with results', () => {
    let ips = ['65.10.123.1', '65.10.123.2', '65.10.123.3']
    let dataArray = ips.map(ip => {
      return { node_ip: ip }
    })
    let result = ips.map(ip => {
      return { public_uri: `http://${ip}` }
    })
    let tx = { data: dataArray }
    tx = JSON.stringify(tx)
    tx = Buffer.from(tx, 'ascii').toString('base64')
    tx = Buffer.from(tx, 'ascii').toString('base64')
    before(() => {
      nodes.setTmRpc({
        getAbciInfo: async () => {
          return { result: { response: { data: '{"last_mint_block":27,"prev_mint_block":27}' } } }
        },
        getTxSearch: async () => {
          return { result: { txs: [tx] } }
        }
      })
    })
    it('should return correct nodes', done => {
      request(insecureServer)
        .get('/nodes/random')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.be.a('array')
            .and.to.deep.equal(result)
          done()
        })
    })
  })
})
