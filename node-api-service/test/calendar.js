/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const calendar = require('../lib/endpoints/calendar.js')

describe('Calendar Controller', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(false)
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('GET /calendar/:txid with bad TM connection', () => {
    before(() => {
      calendar.setTmRpc({
        getTransactionAsync: async () => {
          return { error: true }
        }
      })
    })
    it('should return proper error with TM communication error', done => {
      request(insecureServer)
        .get('/calendar/0xdeadbeef')
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
            .and.to.equal('Could not query for tx by hash')
          done()
        })
    })
  })

  describe('GET /calendar/:txid with unknown txid', () => {
    before(() => {
      calendar.setTmRpc({
        getTransactionAsync: async () => {
          return { error: { responseCode: 404, message: 'notfound' } }
        }
      })
    })
    it('should return proper error with unknown txId', done => {
      request(insecureServer)
        .get('/calendar/0xdeadbeef')
        .expect('Content-type', /json/)
        .expect(404)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('NotFound')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal(`Could not find transaction with id = '0xdeadbeef'`)
          done()
        })
    })
  })

  describe('GET /calendar/:txid', () => {
    let txResult = { tx: { data: 'data', other: 'other' } }
    before(() => {
      calendar.setTmRpc({
        getTransactionAsync: async () => {
          return { result: txResult }
        }
      })
    })
    it('should return proper value', done => {
      request(insecureServer)
        .get('/calendar/0xdeadbeef')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.be.a('object')
            .and.to.deep.equal(txResult)
          done()
        })
    })
  })

  describe('GET /calendar/:txid/data with bad TM connection', () => {
    before(() => {
      calendar.setTmRpc({
        getTransactionAsync: async () => {
          return { error: true }
        }
      })
    })
    it('should return proper error with TM communication error', done => {
      request(insecureServer)
        .get('/calendar/0xdeadbeef/data')
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
            .and.to.equal('Could not query for tx by hash')
          done()
        })
    })
  })

  describe('GET /calendar/:txid/data with unknown txid', () => {
    before(() => {
      calendar.setTmRpc({
        getTransactionAsync: async () => {
          return { error: { responseCode: 404, message: 'notfound' } }
        }
      })
    })
    it('should return proper error with unknown txId', done => {
      request(insecureServer)
        .get('/calendar/0xdeadbeef/data')
        .expect('Content-type', /json/)
        .expect(404)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('NotFound')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal(`Could not find transaction with id = '0xdeadbeef'`)
          done()
        })
    })
  })

  describe('GET /calendar/:txid/data', () => {
    before(() => {
      calendar.setTmRpc({
        getTransactionAsync: async () => {
          return { result: { tx: { data: 'data' } } }
        }
      })
    })
    it('should return proper data value', done => {
      request(insecureServer)
        .get('/calendar/0xdeadbeef/data')
        .expect('Content-type', /text/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.text)
            .to.be.a('string')
            .and.to.equal('data')
          done()
        })
    })
  })
})
