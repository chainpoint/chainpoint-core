/* global describe, it, beforeEach, afterEach */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')

const app = require('../server.js')

describe('Root Controller', () => {
  let insecureServer = null
  beforeEach(async () => {
    app.setThrottle(() => (req, res, next) => next())
    insecureServer = await app.startInsecureRestifyServerAsync(false)
  })
  afterEach(() => {
    insecureServer.close()
  })

  describe('GET /', () => {
    it('should return teapot error', done => {
      request(insecureServer)
        .get('/')
        .expect('Content-type', /json/)
        .expect(418)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('ImATeapot')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('This is an API endpoint. Please consult https://chainpoint.org')
          done()
        })
    })
  })
})
