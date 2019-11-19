/* global describe, it, before, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')
const proofs = require('../lib/endpoints/proofs.js')

describe('Proofs Controller', () => {
  let apiServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    apiServer = await app.startAPIServerAsync(false)
  })
  afterEach(() => {
    apiServer.close()
  })

  describe('GET /proofs with no hashid', () => {
    it('should return proper error', done => {
      request(apiServer)
        .get('/proofs')
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
            .and.to.equal('invalid request, at least one hash id required')
          done()
        })
    })
  })

  describe('GET /proofs with too many hashid', () => {
    let hashids = []
    for (let x = 0; x < 300; x++) {
      hashids.push('hashid')
    }
    hashids = hashids.join(',')
    it('should return proper error', done => {
      request(apiServer)
        .get('/proofs')
        .set({ hashids: hashids })
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
            .and.to.equal('invalid request, too many hash ids (250 max)')
          done()
        })
    })
  })

  describe('GET /proofs with invalid hashid', () => {
    let hashids = []
    for (let x = 0; x < 5; x++) {
      hashids.push('hashid')
    }
    hashids = hashids.join(',')
    it('should return proper error', done => {
      request(apiServer)
        .get('/proofs')
        .set({ hashids: hashids })
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
            .and.to.equal(`invalid request, bad hash_id: hashid`)
          done()
        })
    })
  })

  describe('GET /proofs with db error', () => {
    before(() => {
      proofs.setProof({
        getProofsByProofIdsAsync: () => {
          throw new Error()
        }
      })
    })
    let hashids = 'dbcd35d0-6b77-11e9-9c57-0101a866898d'
    it('should return proper error', done => {
      request(apiServer)
        .get('/proofs')
        .set({ hashids: hashids })
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
            .and.to.equal('error retrieving proofs')
          done()
        })
    })
  })

  describe('GET /proofs with one known and one unknown', () => {
    before(() => {
      proofs.setProof({
        getProofsByProofIdsAsync: () => {
          return [
            { hash_id: 'dbcd35d0-6b77-11e9-9c57-0101a866898d', proof: '{"key0": "value", "key1": 27}' },
            { hash_id: 'ffcd35d0-6b77-11e9-9c57-0101a866898d', proof: null }
          ]
        }
      })
    })
    let hashids = 'dbcd35d0-6b77-11e9-9c57-0101a866898d,ffcd35d0-6b77-11e9-9c57-0101a866898d'
    it('should return proper valid result', done => {
      request(apiServer)
        .get('/proofs')
        .set({ hashids: hashids })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.be.a('array')
          expect(res.body[0])
            .to.have.property('hash_id')
            .and.to.be.a('string')
            .and.to.equal('dbcd35d0-6b77-11e9-9c57-0101a866898d')
          expect(res.body[0])
            .to.have.property('proof')
            .and.to.be.a('object')
          expect(res.body[0].proof)
            .to.have.property('key0')
            .and.to.be.a('string')
            .and.to.equal('value')
          expect(res.body[0].proof)
            .to.have.property('key1')
            .and.to.be.a('number')
            .and.to.equal(27)
          expect(res.body[1])
            .to.have.property('hash_id')
            .and.to.be.a('string')
            .and.to.equal('ffcd35d0-6b77-11e9-9c57-0101a866898d')
          expect(res.body[1])
            .to.have.property('proof')
            .and.to.equal(null)
          done()
        })
    })
  })
})
