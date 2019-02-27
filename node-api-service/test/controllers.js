/* global describe, it */

process.env.NODE_ENV = 'test'

// test related packages
const expect = require('chai').expect
const request = require('supertest')
const crypto = require('crypto')
const uuidTime = require('uuid-time')
const BLAKE2s = require('blake2s-js')

const app = require('../server')
const server = app.server
const hashes = require('../lib/endpoints/hashes')

describe('Home Controller', () => {
  describe('GET /', () => {
    it('should return teapot error', done => {
      request(server)
        .get('/')
        .expect('Content-type', /json/)
        .expect(418)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('ImATeapotError')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('This is an API endpoint. Please consult https://chainpoint.org')
          done()
        })
    })
  })
})

describe('Hashes Controller', () => {
  describe('POST /hashes', () => {
    it('should return proper error with invalid content type', done => {
      request(server)
        .post('/hashes')
        .set('Content-type', 'text/plain')
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
            .and.to.equal('invalid content type')
          done()
        })
    })

    it('should return proper error with missing authorization key', done => {
      request(server)
        .post('/hashes')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: missing authorization key')
          done()
        })
    })

    it('should return proper error with bad authorization value, one string', done => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'qweqwe')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: bad authorization value')
          done()
        })
    })

    it('should return proper error with bad authorization value, no bearer', done => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'qweqwe ababababab')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: bad authorization value')
          done()
        })
    })

    it('should return proper error with bad authorization value, missing tnt-address', done => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: missing tnt-address key')
          done()
        })
    })

    it('should return proper error with bad authorization value, bad tnt-address', done => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0xbad')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: invalid tnt-address value')
          done()
        })
    })

    it('should return proper error with missing hash', done => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ name: 'Manny' })
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
            .and.to.equal('invalid JSON body: missing hash')
          done()
        })
    })

    it('should return proper error with hash not a string', done => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ hash: ['badhash'] })
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
            .and.to.equal('invalid JSON body: bad hash submitted')
          done()
        })
    })

    it('should return proper error with invalid hash', done => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ hash: 'badhash' })
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
            .and.to.equal('invalid JSON body: bad hash submitted')
          done()
        })
    })

    it('should return proper error with no AMQP connection', done => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
        .expect('Content-type', /json/)
        .expect(500)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InternalServerError')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('Message could not be delivered')
          done()
        })
    })

    it('should return proper error with unknown tnt-address', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: unknown tnt-address')
          done()
        })
    })

    it('should return proper error with bad hmac value', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      let tntAddr = '0x1234567890123456789012345678901234567890'

      let hmacKey = crypto.randomBytes(32).toString('hex')
      let hash = crypto.createHmac('sha256', hmacKey)
      let hmac = hash.update('bad').digest('hex')

      request(server)
        .post('/hashes')
        .set('Authorization', `bearer ${hmac}`)
        .set('tnt-address', tntAddr)
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body)
            .to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body)
            .to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: bad hmac value')
          done()
        })
    })

    // TEMP DISABLE WHILE CREDIT CHECK TURNED OFF
    // it('should return proper error with zero balance', (done) => {
    //   app.setAMQPChannel({
    //     sendToQueue: function () { }
    //   })

    //   let tntAddr = '0x1234567890123456789012345678901234567890'

    //   let hmacKey = crypto.randomBytes(32).toString('hex')
    //   let hash = crypto.createHmac('sha256', hmacKey)
    //   let hmac = hash.update(tntAddr).digest('hex')

    //   request(server)
    //     .post('/hashes')
    //     .set('Authorization', `bearer ${hmac}`)
    //     .set('tnt-address', tntAddr)
    //     .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
    //     .expect('Content-type', /json/)
    //     .expect(403)
    //     .end((err, res) => {
    //       expect(err).to.equal(null)
    //       expect(res.body).to.have.property('code')
    //         .and.to.be.a('string')
    //         .and.to.equal('NotAuthorized')
    //       expect(res.body).to.have.property('message')
    //         .and.to.be.a('string')
    //         .and.to.equal('insufficient tntCredit remaining: 0')
    //       done()
    //     })
    // })

    it('should return a matched set of metadata and UUID embedded timestamps', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      let tntAddr = '0x1234567890123456789012345678901234567890'

      let hmacKey = crypto.randomBytes(32).toString('hex')
      let hash = crypto.createHmac('sha256', hmacKey)
      let hmac = hash.update(tntAddr).digest('hex')

      request(server)
        .post('/hashes')
        .set('Authorization', `bearer ${hmac}`)
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('hash_id')
          expect(res.body).to.have.property('submitted_at')
          // The UUID timestamp has ms level precision, ISO8601 only to the second.
          // Check that they are within 1000ms of each other.
          expect(parseInt(uuidTime.v1(res.body.hash_id)) - Date.parse(res.body.submitted_at)).to.be.within(0, 1000)
          done()
        })
    })

    it('should return a v1 UUID node embedded with a partial SHA256 over timestamp and hash', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      let tntAddr = '0x1234567890123456789012345678901234567890'

      let hmacKey = crypto.randomBytes(32).toString('hex')
      let hash = crypto.createHmac('sha256', hmacKey)
      let hmac = hash.update(tntAddr).digest('hex')

      request(server)
        .post('/hashes')
        .set('Authorization', `bearer ${hmac}`)
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('hash_id')
          // Knowing the original hash, the timestamp from the UUID,
          // and the personalization bytes,
          // you should be able to calculate whether the UUID 'Node ID'
          // data segment is the 5 byte BLAKE2s hash of the timestamp
          // embedded in the UUID and the hash submitted to get this UUID.
          let t = parseInt(uuidTime.v1(res.body.hash_id))

          // 5 byte length BLAKE2s hash w/ personalization
          let h = new BLAKE2s(5, { personalization: Buffer.from('CHAINPNT') })
          let hashStr = [t.toString(), t.toString().length, res.body.hash, res.body.hash.length].join(':')

          h.update(Buffer.from(hashStr))
          let shortHashNodeBuf = Buffer.concat([Buffer.from([0x01]), h.digest()])
          // Last segment of UUIDv1 contains BLAKE2s hash to be matched
          expect(res.body.hash_id.split('-')[4]).to.equal(shortHashNodeBuf.toString('hex'))
          done()
        })
    })

    it('should return proper result with valid call', done => {
      app.setAMQPChannel({
        sendToQueue: function() {}
      })

      let tntAddr = '0x1234567890123456789012345678901234567890'

      let hmacKey = crypto.randomBytes(32).toString('hex')
      let hash = crypto.createHmac('sha256', hmacKey)
      let hmac = hash.update(tntAddr).digest('hex')

      request(server)
        .post('/hashes')
        .set('Authorization', `bearer ${hmac}`)
        .set('tnt-address', tntAddr)
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res).to.have.property('body')
          expect(res.body).to.have.property('hash_id')
          expect(res.body)
            .to.have.property('hash')
            .and.to.equal('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12')
          expect(res.body).to.have.property('submitted_at')
          expect(res.body).to.have.property('processing_hints')
          expect(res.body.processing_hints)
            .to.have.property('cal')
            .and.to.be.a('string')
          expect(res.body.processing_hints)
            .to.have.property('eth')
            .and.to.be.a('string')
          expect(res.body.processing_hints)
            .to.have.property('btc')
            .and.to.be.a('string')
          done()
        })
    })
  })
})

describe('Calendar Controller', () => {
  describe('GET /calendar/height', () => {
    it('should return proper error with bad height', done => {
      request(server)
        .get('/calendar/badheight')
        .set('Content-type', 'text/plain')
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
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })
  })

  describe('GET /calendar/height', () => {
    it('should return proper error with negative height', done => {
      request(server)
        .get('/calendar/-1')
        .set('Content-type', 'text/plain')
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
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })
  })

  describe('GET /calendar/height/data', () => {
    it('should return proper error with bad height', done => {
      request(server)
        .get('/calendar/badheight/data')
        .set('Content-type', 'text/plain')
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
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })

    it('should return proper error with negative height', done => {
      request(server)
        .get('/calendar/-2/data')
        .set('Content-type', 'text/plain')
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
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })
  })

  describe('GET /calendar/height/hash', () => {
    it('should return proper error with bad height', done => {
      request(server)
        .get('/calendar/badheight/hash')
        .set('Content-type', 'text/plain')
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
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })

    it('should return proper error with negative height', done => {
      request(server)
        .get('/calendar/-2/hash')
        .set('Content-type', 'text/plain')
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
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })
  })
})

describe('Functions', () => {
  describe('calling generatePostHashResponse with one hash', () => {
    it('should return proper response object', done => {
      let res = hashes.generatePostHashResponse('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12', {
        tntCredit: 9999
      })
      expect(res).to.have.property('hash_id')
      expect(res)
        .to.have.property('hash')
        .and.to.equal('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12')
      expect(res).to.have.property('submitted_at')
      expect(res).to.have.property('processing_hints')
      expect(res.processing_hints)
        .to.have.property('cal')
        .and.to.be.a('string')
      expect(res.processing_hints)
        .to.have.property('eth')
        .and.to.be.a('string')
      expect(res.processing_hints)
        .to.have.property('btc')
        .and.to.be.a('string')
      expect(res)
        .to.have.property('tnt_credit_balance')
        .and.to.equal(9999)
      done()
    })
  })
})
