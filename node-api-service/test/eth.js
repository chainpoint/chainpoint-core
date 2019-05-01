/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const checkWhitelist = require('../lib/middleware/eth-tx-whitelist.js')
const eth = require('../lib/endpoints/eth.js')

describe('Eth Controller', () => {
  let insecureServer = null
  beforeEach(async () => {
    checkWhitelist.setTA('0x684e7D2B54D2fc9fef0138ce00702445cEAd9cEA')
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync()
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('GET /eth/:addr/stats with bad address', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .get('/eth/badaddr/stats')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, invalid ethereum address supplied')
          done()
        })
    })
  })

  describe('GET /eth/:addr/stats with gas price error', () => {
    before(() => {
      eth.setFP({
        getGasPrice: () => {
          throw new Error()
        }
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .get('/eth/0x90f4268c2af35354f1998932c6f360f14211d71a/stats')
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
            .and.to.equal('Error when attempting to retrieve gas price')
          done()
        })
    })
  })

  describe('GET /eth/:addr/stats with nonce error', () => {
    before(() => {
      eth.setFP({
        getGasPrice: () => {
          return { toNumber: () => 20000 }
        },
        getTransactionCount: () => {
          throw new Error()
        }
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .get('/eth/0x90f4268c2af35354f1998932c6f360f14211d71a/stats')
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
            .and.to.equal('Error when attempting to retrieve transaction count')
          done()
        })
    })
  })

  describe('GET /eth/:addr/stats success', () => {
    let ethAddress = '0x90f4268c2af35354f1998932c6f360f14211d71a'
    before(() => {
      eth.setFP({
        getGasPrice: () => {
          return { toNumber: () => 20000 }
        },
        getTransactionCount: () => 234
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .get(`/eth/${ethAddress}/stats`)
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property(ethAddress)
            .and.to.be.a('object')
          expect(res.body[ethAddress])
            .to.have.property('creditPrice')
            .and.to.be.a('number')
          expect(res.body[ethAddress])
            .to.have.property('gasPrice')
            .and.to.be.a('number')
            .and.to.equal(20000)
          expect(res.body[ethAddress])
            .to.have.property('transactionCount')
            .and.to.be.a('number')
            .and.to.equal(234)
          done()
        })
    })
  })

  describe('POST /eth/broadcast with no tx', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, tx must be supplied')
          done()
        })
    })
  })

  describe('POST /eth/broadcast with bad tx', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({ tx: 'deadbeef' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, tx must begin with 0x')
          done()
        })
    })
  })

  describe('POST /eth/broadcast with bad tx no hex', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({ tx: '0xnothexdeadbeef' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, non hex tx value supplied')
          done()
        })
    })
  })

  describe('POST /eth/broadcast with non eth tx', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({ tx: '0xdeadbeefcafe' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, invalid ethereum tx body supplied')
          done()
        })
    })
  })

  describe('POST /eth/broadcast with non chainpoint related eth tx', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894000e7d2b54d2fc9fef0138ce00702445cead9cea80a4bd6ff20b000000000000000000000000000000000000000000000000000000003b9aca001ba04ee96fa3979190df8141c95b960b21f7d48eac705035c4983e4132c43ac35012a05ca005a878e61e405c5191d50c24589567abf279c053b51516ba0f4fc4bfbdb5'
        })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, transaction must interact with Chainpoint token or registry contract')
          done()
        })
    })
  })

  describe('POST /eth/broadcast with send error', () => {
    before(() => {
      eth.setFP({
        sendTransaction: () => {
          throw new Error('senderror')
        }
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4bd6ff20b000000000000000000000000000000000000000000000000000000003b9aca001ca064d1e9fcbd45fb666996232481aea69e12b59982097deb8fe6632a06accf0632a032352440244d001856014b7381e7cf23ec51ef941388d30fabc9beb8fd65d1a8'
        })
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
            .and.to.equal('senderror')
          done()
        })
    })
  })

  describe('POST /eth/broadcast with wait error', () => {
    before(() => {
      eth.setFP({
        sendTransaction: () => {
          return { hash: 'deadbeefcafe' }
        },
        waitForTransaction: () => {
          throw new Error('waiterror')
        }
      })
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4bd6ff20b000000000000000000000000000000000000000000000000000000003b9aca001ca064d1e9fcbd45fb666996232481aea69e12b59982097deb8fe6632a06accf0632a032352440244d001856014b7381e7cf23ec51ef941388d30fabc9beb8fd65d1a8'
        })
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
            .and.to.equal('waiterror')
          done()
        })
    })
  })

  describe('POST /eth/broadcast with success', () => {
    before(() => {
      eth.setFP({
        sendTransaction: () => {
          return { hash: 'deadbeefcafe' }
        },
        waitForTransaction: () => {
          return {
            transactionHash: '0xdeadbeef',
            blockHash: '0xcafe',
            blockNumber: 27,
            gasUsed: { toNumber: () => 35000 }
          }
        }
      })
    })
    it('should return success', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4bd6ff20b000000000000000000000000000000000000000000000000000000003b9aca001ca064d1e9fcbd45fb666996232481aea69e12b59982097deb8fe6632a06accf0632a032352440244d001856014b7381e7cf23ec51ef941388d30fabc9beb8fd65d1a8'
        })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('transactionHash')
            .and.to.equal('0xdeadbeef')
          expect(res.body)
            .to.have.property('blockHash')
            .and.to.equal('0xcafe')
          expect(res.body)
            .to.have.property('blockNumber')
            .and.to.be.a('number')
            .and.to.equal(27)
          expect(res.body)
            .to.have.property('gasUsed')
            .and.to.be.a('number')
            .and.to.equal(35000)
          done()
        })
    })
  })
})
