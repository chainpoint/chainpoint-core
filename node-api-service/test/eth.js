/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const eth = require('../lib/endpoints/eth.js')
const ethers = require('ethers')
const tokenABI = require('./sample_data/tokenABI_tests.js')
const regABI = require('./sample_data/regABI_tests.js')

describe('Eth Controller - Public Mode', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(false)
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

  describe('POST /eth/broadcast with non chainpoint related eth tx, bad tx calls', () => {
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

  describe('POST /eth/broadcast with non chainpoint related eth tx, bad contract addresses', () => {
    before(() => {
      eth.setTA('0xdeadbeef')
      eth.setRA('0xdeadbeef')
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({
          tx:
            '0xf88b82061c8502540be4008302d2a894684e7d2b54d2fc9fef0138ce00702445cead9cea80a4bd6ff20b000000000000000000000000000000000000000000000000000000003b9aca001ca064d1e9fcbd45fb666996232481aea69e12b59982097deb8fe6632a06accf0632a032352440244d001856014b7381e7cf23ec51ef941388d30fabc9beb8fd65d1a8'
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

  describe('POST /eth/broadcast with unsupported token method', () => {
    before(() => {
      eth.setTA('0x0Cc0ADFb92B45195bA844945E9d69361cB0529a3')
      eth.setTCI(new ethers.utils.Interface(tokenABI))
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({
          tx:
            '0xf8aa038502540be4008302d2a8940cc0adfb92b45195ba844945e9d69361cb0529a380b844a9059cbb0000000000000000000000003a8264f138489f80d9cca443c3a534b73f4b6401000000000000000000000000000000000000000000000000000000746a5288001ba0d557f0f5c6b8f90cd20972d0a60d77094d9c8f8d635d158ef38c5f0709a46272a01281e47680298300f352c9c039a84b8945bf8d6b8a973c8cdf3650a0807c25fb'
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
            .and.to.equal(`invalid request, transaction may only call 'approve' method(s) on that contract`)
          done()
        })
    })
  })

  describe('POST /eth/broadcast with unsupported reg method', () => {
    before(() => {
      eth.setRA('0x3a8264f138489f80D9CcA443C3A534B73F4B6401')
      eth.setRCI(new ethers.utils.Interface(regABI))
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({
          tx:
            '0xf869038502540be4008302d2a8943a8264f138489f80d9cca443c3a534b73f4b640180848456cb591ba0c8ece996fd486d501db127f18478965c0a4a1782fcdd7af0f1416d8dbaa5d6a9a006d494bc817c14cca1f1d883b1e21c80efb1c0aa0e349f685b5aa21b2ae8441f'
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
            .and.to.equal(
              `invalid request, transaction may only call 'stake,unStake,updateStake' method(s) on that contract`
            )
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
      eth.setTA('0x0Cc0ADFb92B45195bA844945E9d69361cB0529a3')
      eth.setTCI(new ethers.utils.Interface(tokenABI))
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({
          tx:
            '0xf8aa038502540be4008302d2a8940cc0adfb92b45195ba844945e9d69361cb0529a380b844095ea7b30000000000000000000000003a8264f138489f80d9cca443c3a534b73f4b6401000000000000000000000000000000000000000000000000000000746a5288001ba0aaf904fd07752d48a178a66ca533ad291dae487c4b400c78b4428f583929f9fba0637b141375e2370a407eb76b6a99b14de0f2edfefbfaad7dfdff4a9109c63ad9'
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
      eth.setTA('0x0Cc0ADFb92B45195bA844945E9d69361cB0529a3')
      eth.setTCI(new ethers.utils.Interface(tokenABI))
    })
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({
          tx:
            '0xf8aa038502540be4008302d2a8940cc0adfb92b45195ba844945e9d69361cb0529a380b844095ea7b30000000000000000000000003a8264f138489f80d9cca443c3a534b73f4b6401000000000000000000000000000000000000000000000000000000746a5288001ba0aaf904fd07752d48a178a66ca533ad291dae487c4b400c78b4428f583929f9fba0637b141375e2370a407eb76b6a99b14de0f2edfefbfaad7dfdff4a9109c63ad9'
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
      eth.setTA('0x0Cc0ADFb92B45195bA844945E9d69361cB0529a3')
      eth.setTCI(new ethers.utils.Interface(tokenABI))
    })
    it('should return success', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .send({
          tx:
            '0xf8aa038502540be4008302d2a8940cc0adfb92b45195ba844945e9d69361cb0529a380b844095ea7b30000000000000000000000003a8264f138489f80d9cca443c3a534b73f4b6401000000000000000000000000000000000000000000000000000000746a5288001ba0aaf904fd07752d48a178a66ca533ad291dae487c4b400c78b4428f583929f9fba0637b141375e2370a407eb76b6a99b14de0f2edfefbfaad7dfdff4a9109c63ad9'
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

describe('Eth Controller - Private Mode', () => {
  let insecureServer = null
  beforeEach(async () => {
    eth.setTA('0x684e7D2B54D2fc9fef0138ce00702445cEAd9cEA')
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(true)
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('GET /eth/:addr/stats', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .get('/eth/badaddr/stats')
        .expect('Content-type', /json/)
        .expect(404)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('ResourceNotFound')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('/eth/badaddr/stats does not exist')
          done()
        })
    })
  })

  describe('POST /eth/broadcast', () => {
    it('should return proper error', done => {
      request(insecureServer)
        .post('/eth/broadcast')
        .expect('Content-type', /json/)
        .expect(404)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('ResourceNotFound')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('/eth/broadcast does not exist')
          done()
        })
    })
  })
})
