/* global describe, it */

process.env.NODE_ENV = 'test'
process.env.MIN_NODE_VERSION_EXISTING = '1.2.0'
process.env.MIN_NODE_VERSION_NEW = '1.2.0'
process.env.MIN_TNT_GRAINS_BALANCE_FOR_REWARD = 500000000000

// test related packages
const expect = require('chai').expect
const request = require('supertest')
const crypto = require('crypto')
const moment = require('moment')
const uuidTime = require('uuid-time')
const BLAKE2s = require('blake2s-js')
const Charlatan = require('charlatan')

const app = require('../server')
const server = app.server
const hashes = require('../lib/endpoints/hashes')

app.setRedis({
  hgetall: (key) => { return null },
  hmset: (key, value) => { return null },
  expire: (key, ms) => { return null },
  set: (key) => { return null }
})

app.setMinNodeVersionNew('1.2.0')
app.setMinNodeVersionExisting('1.2.0')

describe('Home Controller', () => {
  describe('GET /', () => {
    it('should return teapot error', (done) => {
      request(server)
        .get('/')
        .expect('Content-type', /json/)
        .expect(418)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('ImATeapotError')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('This is an API endpoint. Please consult https://chainpoint.org')
          done()
        })
    })
  })
})

describe('Hashes Controller', () => {
  describe('POST /hashes', () => {
    it('should return proper error with invalid content type', (done) => {
      request(server)
        .post('/hashes')
        .set('Content-type', 'text/plain')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid content type')
          done()
        })
    })

    it('should return proper error with missing authorization key', (done) => {
      request(server)
        .post('/hashes')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: missing authorization key')
          done()
        })
    })

    it('should return proper error with bad authorization value, one string', (done) => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'qweqwe')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: bad authorization value')
          done()
        })
    })

    it('should return proper error with bad authorization value, no bearer', (done) => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'qweqwe ababababab')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: bad authorization value')
          done()
        })
    })

    it('should return proper error with bad authorization value, missing tnt-address', (done) => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: missing tnt-address key')
          done()
        })
    })

    it('should return proper error with bad authorization value, bad tnt-address', (done) => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0xbad')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: invalid tnt-address value')
          done()
        })
    })

    it('should return proper error with missing hash', (done) => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ name: 'Manny' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body: missing hash')
          done()
        })
    })

    it('should return proper error with hash not a string', (done) => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ hash: ['badhash'] })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body: bad hash submitted')
          done()
        })
    })

    it('should return proper error with invalid hash', (done) => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ hash: 'badhash' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body: bad hash submitted')
          done()
        })
    })

    it('should return proper error with no AMQP connection', (done) => {
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
        .expect('Content-type', /json/)
        .expect(500)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InternalServerError')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('Message could not be delivered')
          done()
        })
    })

    it('should return proper error with NTP < NIST value', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })
      app.setNistLatest('3002759084:8E00C0AF2B68E33CC453BF45A1689A6804700C083478FEB34E4694422999B6F745C2F837D7BA983F9D7BA52F7CC62965B8E1B7384CD8177003B5D3A0D099D93C')
      request(server)
        .post('/hashes')
        .set('Authorization', 'bearer ababab121212')
        .set('tnt-address', '0x1234567890123456789012345678901234567890')
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
        .expect('Content-type', /json/)
        .expect(500)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InternalServerError')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('Bad NTP time')
          done()
        })
    })

    it('should return proper error with unknown tnt-address', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      app.setNistLatest('1400585240:8E00C0AF2B68E33CC453BF45A1689A6804700C083478FEB34E4694422999B6F745C2F837D7BA983F9D7BA52F7CC62965B8E1B7384CD8177003B5D3A0D099D93C')
      app.setHashesRegisteredNode({
        findOne: (params) => {
          return null
        }
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
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('authorization denied: unknown tnt-address')
          done()
        })
    })

    it('should return proper error with bad hmac value', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      let tntAddr = '0x1234567890123456789012345678901234567890'

      let hmacKey = crypto.randomBytes(32).toString('hex')
      let hash = crypto.createHmac('sha256', hmacKey)
      let hmac = hash.update('bad').digest('hex')

      app.setNistLatest('1400585240:8E00C0AF2B68E33CC453BF45A1689A6804700C083478FEB34E4694422999B6F745C2F837D7BA983F9D7BA52F7CC62965B8E1B7384CD8177003B5D3A0D099D93C')
      app.setHashesRegisteredNode({
        findOne: (params) => {
          return {
            tntAddr: tntAddr,
            hmacKey: hmacKey,
            tntCredit: 10
          }
        }
      })
      request(server)
        .post('/hashes')
        .set('Authorization', `bearer ${hmac}`)
        .set('tnt-address', tntAddr)
        .send({ hash: 'ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12' })
        .expect('Content-type', /json/)
        .expect(401)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidCredentials')
          expect(res.body).to.have.property('message')
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

    //   app.setNistLatest('1400585240:8E00C0AF2B68E33CC453BF45A1689A6804700C083478FEB34E4694422999B6F745C2F837D7BA983F9D7BA52F7CC62965B8E1B7384CD8177003B5D3A0D099D93C')
    //   app.setHashesRegisteredNode({
    //     findOne: (params) => {
    //       return {
    //         tntAddr: tntAddr,
    //         hmacKey: hmacKey,
    //         tntCredit: 0
    //       }
    //     }
    //   })
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

    it('should return a matched set of metadata and UUID embedded timestamps', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      let tntAddr = '0x1234567890123456789012345678901234567890'

      let hmacKey = crypto.randomBytes(32).toString('hex')
      let hash = crypto.createHmac('sha256', hmacKey)
      let hmac = hash.update(tntAddr).digest('hex')

      app.setNistLatest('1400585240:8E00C0AF2B68E33CC453BF45A1689A6804700C083478FEB34E4694422999B6F745C2F837D7BA983F9D7BA52F7CC62965B8E1B7384CD8177003B5D3A0D099D93C')
      app.setHashesRegisteredNode({
        findOne: (params) => {
          return {
            tntAddr: tntAddr,
            hmacKey: hmacKey,
            tntCredit: 10,
            decrement: (params) => { }
          }
        }
      })
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

    it('should return a v1 UUID node embedded with a partial SHA256 over timestamp and hash', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      let tntAddr = '0x1234567890123456789012345678901234567890'

      let hmacKey = crypto.randomBytes(32).toString('hex')
      let hash = crypto.createHmac('sha256', hmacKey)
      let hmac = hash.update(tntAddr).digest('hex')

      app.setNistLatest('1400585240:8E00C0AF2B68E33CC453BF45A1689A6804700C083478FEB34E4694422999B6F745C2F837D7BA983F9D7BA52F7CC62965B8E1B7384CD8177003B5D3A0D099D93C')
      app.setHashesRegisteredNode({
        findOne: (params) => {
          return {
            tntAddr: tntAddr,
            hmacKey: hmacKey,
            tntCredit: 10,
            decrement: (params) => { }
          }
        }
      })
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
          // Knowing the original hash, the timestamp from the UUID, the
          // latest available NIST data, and the personalization bytes,
          // you should be able to calculate whether the UUID 'Node ID'
          // data segment is the 5 byte BLAKE2s hash of the timestamp
          // embedded in the UUID and the hash submitted to get this UUID.
          let t = parseInt(uuidTime.v1(res.body.hash_id))

          // 5 byte length BLAKE2s hash w/ personalization
          let h = new BLAKE2s(5, { personalization: Buffer.from('CHAINPNT') })
          let hashStr = [
            t.toString(),
            t.toString().length,
            res.body.hash,
            res.body.hash.length,
            res.body.nist,
            res.body.nist.length
          ].join(':')

          h.update(Buffer.from(hashStr))
          let shortHashNodeBuf = Buffer.concat([Buffer.from([0x01]), h.digest()])
          // Last segment of UUIDv1 contains BLAKE2s hash to be matched
          expect(res.body.hash_id.split('-')[4]).to.equal(shortHashNodeBuf.toString('hex'))
          done()
        })
    })

    it('should return proper result with valid call', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      let tntAddr = '0x1234567890123456789012345678901234567890'

      let hmacKey = crypto.randomBytes(32).toString('hex')
      let hash = crypto.createHmac('sha256', hmacKey)
      let hmac = hash.update(tntAddr).digest('hex')

      app.setNistLatest('1400585240:8E00C0AF2B68E33CC453BF45A1689A6804700C083478FEB34E4694422999B6F745C2F837D7BA983F9D7BA52F7CC62965B8E1B7384CD8177003B5D3A0D099D93C')
      app.setHashesRegisteredNode({
        findOne: (params) => {
          return {
            tntAddr: tntAddr,
            hmacKey: hmacKey,
            tntCredit: 10,
            decrement: (params) => { }
          }
        }
      })
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
          expect(res.body).to.have.property('hash').and.to.equal('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12')
          expect(res.body).to.have.property('nist')
          expect(res.body).to.have.property('submitted_at')
          expect(res.body).to.have.property('processing_hints')
          expect(res.body.processing_hints).to.have.property('cal').and.to.be.a('string')
          expect(res.body.processing_hints).to.have.property('eth').and.to.be.a('string')
          expect(res.body.processing_hints).to.have.property('btc').and.to.be.a('string')
          done()
        })
    })
  })
})

describe('Calendar Controller', () => {
  describe('GET /calendar/height', () => {
    it('should return proper error with bad height', (done) => {
      request(server)
        .get('/calendar/badheight')
        .set('Content-type', 'text/plain')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })
  })

  describe('GET /calendar/height', () => {
    it('should return proper error with negative height', (done) => {
      request(server)
        .get('/calendar/-1')
        .set('Content-type', 'text/plain')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })
  })

  describe('GET /calendar/height/data', () => {
    it('should return proper error with bad height', (done) => {
      request(server)
        .get('/calendar/badheight/data')
        .set('Content-type', 'text/plain')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })

    it('should return proper error with negative height', (done) => {
      request(server)
        .get('/calendar/-2/data')
        .set('Content-type', 'text/plain')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })
  })

  describe('GET /calendar/height/hash', () => {
    it('should return proper error with bad height', (done) => {
      request(server)
        .get('/calendar/badheight/hash')
        .set('Content-type', 'text/plain')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })

    it('should return proper error with negative height', (done) => {
      request(server)
        .get('/calendar/-2/hash')
        .set('Content-type', 'text/plain')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid request, height must be a positive integer')
          done()
        })
    })
  })
})

/* TODO: Re-enable using cachedAuditChallenge
describe('Config Controller', () => {
  app.config.setAuditChallenge({
    findOne: async () => {
      return {
        time: 1504898081430,
        minBlock: 9661,
        maxBlock: 10483,
        nonce: 'd9a52b6e1e4cdc46d03b58c6b4b58a01e0eb7b252a83ee5346314a1240561c4d',
        solution: '57d17352247cbbdd2551d5b2401c85c54cb47e92265ac034ada2577cb00f012d'
      }
    }
  })
  app.config.setCalendarBlock({
    findOne: async () => {
      return { id: 27272 }
    }
  })
  describe('GET /config', () => {
    it('should return proper config object', (done) => {
      request(server)
        .get('/config')
        .set('Content-type', 'text/plain')
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('chainpoint_core_base_uri').and.to.equal('http://test.chainpoint.org')
          expect(res.body).to.have.property('anchor_btc')
          expect(res.body).to.have.property('anchor_eth')
          expect(res.body).to.have.property('public_keys')
          expect(res.body).to.have.property('calendar')
          expect(res.body.calendar).to.have.property('height')
          expect(res.body.calendar).to.have.property('audit_challenge').and.to.equal('1504898081430:9661:10483:d9a52b6e1e4cdc46d03b58c6b4b58a01e0eb7b252a83ee5346314a1240561c4d')
          expect(res.body).to.have.property('core_eth_address')
          done()
        })
    })
  })
}) */

describe('Nodes Controller', () => {
  describe('POST /nodes', () => {
    it('should return proper error with invalid content type', (done) => {
      request(server)
        .post('/nodes')
        .set('Content-type', 'text/plain')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid content type')
          done()
        })
    })

    it('should return error with missing node version', (done) => {
      request(server)
        .post('/nodes')
        .send({ public_uri: 'http://65.198.32.187' })
        .expect('Content-type', /json/)
        .expect(426)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('UpgradeRequiredError')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal(`Node version ${process.env.MIN_NODE_VERSION_NEW} or greater required`)
          done()
        })
    })

    it('should return error with bad node version', (done) => {
      request(server)
        .post('/nodes')
        .set('X-Node-Version', 'bad+version')
        .send({ public_uri: 'http://65.198.32.187' })
        .expect('Content-type', /json/)
        .expect(426)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('UpgradeRequiredError')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal(`Node version ${process.env.MIN_NODE_VERSION_NEW} or greater required`)
          done()
        })
    })

    it('should return error with low node version', (done) => {
      request(server)
        .post('/nodes')
        .set('X-Node-Version', '1.1.9')
        .send({ public_uri: 'http://65.198.32.187' })
        .expect('Content-type', /json/)
        .expect(426)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('UpgradeRequiredError')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal(`Node version ${process.env.MIN_NODE_VERSION_NEW} or greater required`)
          done()
        })
    })

    it('should return error with no tnt_addr', (done) => {
      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ public_uri: 'http://65.198.32.187' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body, missing tnt_addr')
          done()
        })
    })

    it('should return error with empty tnt_addr', (done) => {
      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: '', public_uri: 'http://65.198.32.187' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body, empty tnt_addr')
          done()
        })
    })

    it('should return error with malformed tnt_addr', (done) => {
      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: '0xabc', public_uri: 'http://65.198.32.187' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body, malformed tnt_addr')
          done()
        })
    })

    it('should return error with bad public_uri', (done) => {
      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: '0x' + crypto.randomBytes(20).toString('hex'), public_uri: 'badval' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body, invalid public_uri')
          done()
        })
    })

    it('should return error with non-IP public_uri', (done) => {
      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: '0x' + crypto.randomBytes(20).toString('hex'), public_uri: 'http://www.chainpoint.org' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('public_uri hostname must be an IP')
          done()
        })
    })

    it('should return error with private public_uri', (done) => {
      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: '0x' + crypto.randomBytes(20).toString('hex'), public_uri: 'http://127.0.0.1' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('public_uri hostname must not be a private IP')
          done()
        })
    })

    it('should return error with 0.0.0.0 public_uri', (done) => {
      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: '0x' + crypto.randomBytes(20).toString('hex'), public_uri: 'http://0.0.0.0' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('0.0.0.0 not allowed in public_uri')
          done()
        })
    })

    it('should return error if a tnt_addr already exists', (done) => {
      let publicUri1 = 'http://' + Charlatan.Internet.IPv4()
      let publicUri2 = 'http://' + Charlatan.Internet.IPv4()
      let tntAddr1 = '0x' + crypto.randomBytes(20).toString('hex')

      let data = []

      app.setNodesRegisteredNode({
        findOne: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let result = null
          result = data.find((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          })
          return result
        },
        create: (params) => {
          let row = {
            tntAddr: params.tntAddr,
            publicUri: params.publicUri,
            hmacKey: crypto.randomBytes(32).toString('hex')
          }
          data.push(row)
          return row
        }
      })

      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: tntAddr1, public_uri: publicUri1 })
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          request(server)
            .post('/nodes')
            .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
            .send({ tnt_addr: tntAddr1, public_uri: publicUri2 })
            .expect(409)
            .end((err, res) => {
              expect(err).to.equal(null)
              expect(res.body).to.have.property('code')
                .and.to.be.a('string')
                .and.to.equal('ConflictError')
              expect(res.body).to.have.property('message')
                .and.to.be.a('string')
                .and.to.equal('the Ethereum address provided is already registered')
              done()
            })
        })
    })

    it('should not allow public_uri to be registered twice', (done) => {
      let publicUri = 'http://' + Charlatan.Internet.IPv4()
      let tntAddr1 = '0x' + crypto.randomBytes(20).toString('hex')
      let tntAddr2 = '0x' + crypto.randomBytes(20).toString('hex')

      let data = []

      app.setNodesRegisteredNode({
        findOne: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let result = null
          result = data.find((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          })
          return result
        },
        create: (params) => {
          let row = {
            tntAddr: params.tntAddr,
            publicUri: params.publicUri,
            hmacKey: crypto.randomBytes(32).toString('hex')
          }
          data.push(row)
          return row
        }
      })

      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: tntAddr1, public_uri: publicUri })
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          request(server)
            .post('/nodes')
            .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
            .send({ tnt_addr: tntAddr2, public_uri: publicUri })
            .expect(409)
            .end((err, res) => {
              expect(err).to.equal(null)
              expect(res.body).to.have.property('code')
                .and.to.be.a('string')
                .and.to.equal('ConflictError')
              expect(res.body).to.have.property('message')
                .and.to.be.a('string')
                .and.to.equal('the public URI provided is already registered')
              done()
            })
        })
    })

    it('should return OK for valid request', (done) => {
      let publicUri = 'http://' + Charlatan.Internet.IPv4()
      let tntAddr1 = '0x' + crypto.randomBytes(20).toString('hex')
      let hmacKey = crypto.randomBytes(32).toString('hex')

      let data = []

      app.setNodesRegisteredNode({
        findOne: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let result = null
          result = data.find((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          })
          return result
        },
        create: (params) => {
          let row = {
            tntAddr: params.tntAddr,
            publicUri: params.publicUri,
            hmacKey: hmacKey
          }
          data.push(row)
          return row
        }
      })

      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: tntAddr1, public_uri: publicUri })
        .expect('Content-type', /json/)
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('tnt_addr').and.to.equal(tntAddr1)
          expect(res.body).to.have.property('public_uri').and.to.equal(publicUri)
          expect(res.body).to.have.property('hmac_key').and.to.equal(hmacKey)
          done()
        })
    })
  })

  describe('PUT /nodes', () => {
    it('should return proper error with invalid content type', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('Content-type', 'text/plain')
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid content type')
          done()
        })
    })

    it('should return error with missing node version', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('Content-type', 'application/json')
        .expect('Content-type', /json/)
        .expect(426)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('UpgradeRequiredError')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal(`Node version ${process.env.MIN_NODE_VERSION_EXISTING} or greater required`)
          done()
        })
    })

    it('should return error with bad node version', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('X-Node-Version', 'bad+version')
        .set('Content-type', 'application/json')
        .expect('Content-type', /json/)
        .expect(426)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('UpgradeRequiredError')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal(`Node version ${process.env.MIN_NODE_VERSION_EXISTING} or greater required`)
          done()
        })
    })

    it('should return error with low node version', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('X-Node-Version', '1.1.1')
        .set('Content-type', 'application/json')
        .expect('Content-type', /json/)
        .expect(426)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('UpgradeRequiredError')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal(`Node version ${process.env.MIN_NODE_VERSION_EXISTING} or greater required`)
          done()
        })
    })

    it('should return error with malformed tnt_addr', (done) => {
      let randTntAddr = '0xzxczxc'
      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 100000000000 })
      request(server)
        .put('/nodes/' + randTntAddr)
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
        .send({ public_uri: 'http://65.198.32.187' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body, malformed tnt_addr')
          done()
        })
    })

    it('should return error with missing hmac', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')
      let publicUri = 'http://65.198.32.187'
      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
        .send({ public_uri: publicUri })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body, missing hmac')
          done()
        })
    })

    it('should return error with empty hmac', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')
      let publicUri = 'http://65.198.32.187'
      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
        .send({ public_uri: publicUri, hmac: '' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body, empty hmac')
          done()
        })
    })

    it('should return error with invalid hmac', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')
      let publicUri = 'http://65.198.32.187'
      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
        .send({ public_uri: publicUri, hmac: '!badhmac' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body, invalid hmac')
          done()
        })
    })

    it('should return error with invalid hmac', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')
      let publicUri = 'http://65.198.32.187'
      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
        .send({ public_uri: publicUri, hmac: '!badhmac' })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body, invalid hmac')
          done()
        })
    })

    it('should return error with malformed public_uri', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')
      let publicUri = 'baduri'
      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
        .send({ public_uri: publicUri, hmac: crypto.randomBytes(32).toString('hex') })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('invalid JSON body, invalid public_uri')
          done()
        })
    })

    it('should return error with non-IP public_uri', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')
      let publicUri = 'http://www.chainpoint.org'
      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
        .send({ public_uri: publicUri, hmac: crypto.randomBytes(32).toString('hex') })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('public_uri hostname must be an IP')
          done()
        })
    })

    it('should return error with private public_uri', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')
      let publicUri = 'http://127.0.0.1'
      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
        .send({ public_uri: publicUri, hmac: crypto.randomBytes(32).toString('hex') })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('public_uri hostname must not be a private IP')
          done()
        })
    })

    it('should return error with 0.0.0.0 public_uri', (done) => {
      let randTntAddr = '0x' + crypto.randomBytes(20).toString('hex')
      let publicUri = 'http://0.0.0.0'
      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .put('/nodes/' + randTntAddr)
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
        .send({ public_uri: publicUri, hmac: crypto.randomBytes(32).toString('hex') })
        .expect('Content-type', /json/)
        .expect(409)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('InvalidArgument')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('0.0.0.0 not allowed in public_uri')
          done()
        })
    })

    it('should return error for node registration not found', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      let publicUri1 = 'http://65.198.32.187'
      let tntAddr1 = '0x' + crypto.randomBytes(20).toString('hex')
      let hmacKey = crypto.randomBytes(32).toString('hex')

      let data = []

      app.setNodesRegisteredNode({
        findAll: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let results = []
          results = data.filter((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          }).map((item) => {
            item.save = () => { }
            return item
          })
          return results
        },
        create: (params) => {
          let row = {
            tntAddr: params.tntAddr,
            publicUri: params.publicUri,
            hmacKey: hmacKey
          }
          data.push(row)
          return row
        }
      })

      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .put('/nodes/' + tntAddr1)
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
        .send({ public_uri: publicUri1, hmac: crypto.randomBytes(32).toString('hex') })
        .expect('Content-type', /json/)
        .expect(404)
        .end((err, res) => {
          expect(err).to.equal(null)
          expect(res.body).to.have.property('code')
            .and.to.be.a('string')
            .and.to.equal('NotFoundError')
          expect(res.body).to.have.property('message')
            .and.to.be.a('string')
            .and.to.equal('could not find registered Node')
          done()
        })
    })

    it('should return error for bad hmac', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      let publicUri1 = 'http://65.198.32.187'
      let tntAddr1 = '0x' + crypto.randomBytes(20).toString('hex')
      let hmacKey = crypto.randomBytes(32).toString('hex')

      let data = []

      app.setNodesRegisteredNode({
        findAll: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let results = []
          results = data.filter((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          }).map((item) => {
            item.save = () => { }
            return item
          })
          return results
        },
        findOne: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let result = null
          result = data.find((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          })
          return result
        },
        create: (params) => {
          let row = {
            tntAddr: params.tntAddr,
            publicUri: params.publicUri,
            hmacKey: hmacKey
          }
          data.push(row)
          return row
        }
      })

      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: tntAddr1, public_uri: publicUri1 })
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)

          request(server)
            .put('/nodes/' + tntAddr1)
            .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
            .send({ public_uri: publicUri1, hmac: crypto.randomBytes(32).toString('hex') })
            .expect('Content-type', /json/)
            .expect(409)
            .end((err, res) => {
              expect(err).to.equal(null)
              expect(res.body).to.have.property('code')
                .and.to.be.a('string')
                .and.to.equal('InvalidArgument')
              expect(res.body).to.have.property('message')
                .and.to.be.a('string')
                .and.to.equal('Invalid authentication HMAC provided - Try NTP sync')
              done()
            })
        })
    })

    it('should return error for low balance', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      let publicUri = 'http://65.198.32.187'
      let tntAddr1 = '0x' + crypto.randomBytes(20).toString('hex')
      let hmacKey = crypto.randomBytes(32).toString('hex')

      let data = []

      app.setNodesRegisteredNode({
        findAll: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let results = []
          results = data.filter((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          }).map((item) => {
            item.save = () => { }
            return item
          })
          return results
        },
        findOne: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let result = null
          result = data.find((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          })
          return result
        },
        create: (params) => {
          let row = {
            tntAddr: params.tntAddr,
            publicUri: params.publicUri,
            hmacKey: hmacKey
          }
          data.push(row)
          return row
        }
      })

      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: tntAddr1, public_uri: publicUri })
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)

          // HMAC-SHA256(hmac-key, TNT_ADDRESS|IP|YYYYMMDDHHMM)
          let hash = crypto.createHmac('sha256', res.body.hmac_key)
          let formattedDate = moment().utc().format('YYYYMMDDHHmm')
          let hmacTxt = [tntAddr1, publicUri, formattedDate].join('')
          let calculatedHMAC = hash.update(hmacTxt).digest('hex')

          app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 100000000000 })
          request(server)
            .put('/nodes/' + tntAddr1)
            .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
            .send({ public_uri: publicUri, hmac: calculatedHMAC })
            .expect('Content-type', /json/)
            .expect(403)
            .end((err, res) => {
              expect(err).to.equal(null)
              expect(res.body).to.have.property('code')
                .and.to.be.a('string')
                .and.to.equal('ForbiddenError')
              expect(res.body).to.have.property('message')
                .and.to.be.a('string')
                .and.to.equal(`TNT address ${tntAddr1} does not have the minimum balance of ${process.env.MIN_TNT_GRAINS_BALANCE_FOR_REWARD / 100000000} TNT for Node operation`)
              done()
            })
        })
    })

    it('should return OK for valid PUT no change to tnt and updated IP', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      let publicUri1 = 'http://65.198.32.187'
      let publicUri2 = 'http://65.198.32.188'
      let tntAddr1 = '0x' + crypto.randomBytes(20).toString('hex')
      let hmacKey = crypto.randomBytes(32).toString('hex')

      let data = []

      app.setNodesRegisteredNode({
        findAll: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let results = []
          results = data.filter((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          }).map((item) => {
            item.save = () => { }
            return item
          })
          return results
        },
        findOne: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let result = null
          result = data.find((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          })
          return result
        },
        create: (params) => {
          let row = {
            tntAddr: params.tntAddr,
            publicUri: params.publicUri,
            hmacKey: hmacKey
          }
          data.push(row)
          return row
        }
      })

      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: tntAddr1, public_uri: publicUri1 })
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          // HMAC-SHA256(hmac-key, TNT_ADDRESS|IP|YYYYMMDDHHMM)
          let hash = crypto.createHmac('sha256', res.body.hmac_key)
          let formattedDate = moment().utc().format('YYYYMMDDHHmm')
          let hmacTxt = [tntAddr1, publicUri2, formattedDate].join('')
          let calculatedHMAC = hash.update(hmacTxt).digest('hex')

          request(server)
            .put('/nodes/' + tntAddr1)
            .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
            .send({ public_uri: publicUri2, hmac: calculatedHMAC })
            .expect('Content-type', /json/)
            .expect(200)
            .end((err, res) => {
              expect(err).to.equal(null)
              expect(res.body).to.have.property('tnt_addr')
                .and.to.equal(tntAddr1)
              expect(res.body).to.have.property('public_uri')
                .and.to.equal(publicUri2)
              done()
            })
        })
    })

    it('should return OK for valid PUT no change to tnt and updated IP', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      let publicUri1 = 'http://65.198.32.187'
      let publicUri2 = 'http://65.198.32.188'
      let tntAddr1 = '0x' + crypto.randomBytes(20).toString('hex')
      let hmacKey = crypto.randomBytes(32).toString('hex')

      let data = []

      app.setNodesRegisteredNode({
        findAll: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let results = []
          results = data.filter((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          }).map((item) => {
            item.save = () => { }
            return item
          })
          return results
        },
        findOne: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let result = null
          result = data.find((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          })
          return result
        },
        create: (params) => {
          let row = {
            tntAddr: params.tntAddr,
            publicUri: params.publicUri,
            hmacKey: hmacKey
          }
          data.push(row)
          return row
        }
      })

      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: tntAddr1, public_uri: publicUri1 })
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          // HMAC-SHA256(hmac-key, TNT_ADDRESS|IP|YYYYMMDDHHMM)
          let hash = crypto.createHmac('sha256', res.body.hmac_key)
          let formattedDate = moment().utc().format('YYYYMMDDHHmm')
          let hmacTxt = [tntAddr1, publicUri2, formattedDate].join('')
          let calculatedHMAC = hash.update(hmacTxt).digest('hex')

          request(server)
            .put('/nodes/' + tntAddr1)
            .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
            .send({ public_uri: publicUri2, hmac: calculatedHMAC })
            .expect('Content-type', /json/)
            .expect(200)
            .end((err, res) => {
              expect(err).to.equal(null)
              expect(res.body).to.have.property('tnt_addr')
                .and.to.equal(tntAddr1)
              expect(res.body).to.have.property('public_uri')
                .and.to.equal(publicUri2)
              done()
            })
        })
    })

    it('should return OK for valid PUT no change to tnt and removed IP', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      let publicUri1 = 'http://65.198.32.187'
      let tntAddr1 = '0x' + crypto.randomBytes(20).toString('hex')
      let hmacKey = crypto.randomBytes(32).toString('hex')

      let data = []

      app.setNodesRegisteredNode({
        findAll: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let results = []
          results = data.filter((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          }).map((item) => {
            item.save = () => { }
            return item
          })
          return results
        },
        findOne: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let result = null
          result = data.find((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          })
          return result
        },
        create: (params) => {
          let row = {
            tntAddr: params.tntAddr,
            publicUri: params.publicUri,
            hmacKey: hmacKey
          }
          data.push(row)
          return row
        }
      })

      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: tntAddr1, public_uri: publicUri1 })
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)
          // HMAC-SHA256(hmac-key, TNT_ADDRESS|IP|YYYYMMDDHHMM)
          let hash = crypto.createHmac('sha256', res.body.hmac_key)
          let formattedDate = moment().utc().format('YYYYMMDDHHmm')
          let hmacTxt = [tntAddr1, '', formattedDate].join('')
          let calculatedHMAC = hash.update(hmacTxt).digest('hex')

          request(server)
            .put('/nodes/' + tntAddr1)
            .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
            .send({ hmac: calculatedHMAC })
            .expect('Content-type', /json/)
            .expect(200)
            .end((err, res) => {
              expect(err).to.equal(null)
              expect(res.body).to.have.property('tnt_addr')
                .and.to.equal(tntAddr1)
              expect(res.body).to.not.have.property('public_uri')
              done()
            })
        })
    })

    it('should return error for valid PUT no change to tnt and in use IP', (done) => {
      app.setAMQPChannel({
        sendToQueue: function () { }
      })

      let publicUri1 = 'http://65.198.32.187'
      let publicUri2 = 'http://65.198.32.188'
      let tntAddr1 = '0x' + crypto.randomBytes(20).toString('hex')
      let tntAddr2 = '0x' + crypto.randomBytes(20).toString('hex')
      let hmacKey = crypto.randomBytes(32).toString('hex')

      let data = []

      app.setNodesRegisteredNode({
        findAll: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let results = []
          results = data.filter((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          }).map((item) => {
            item.save = () => { }
            return item
          })
          return results
        },
        findOne: (params) => {
          let symbols = Object.getOwnPropertySymbols(params.where)
          let hasOr = symbols.length > 0
          let findParams = []
          if (!hasOr) {
            findParams.push(params.where)
          } else {
            for (let param of params.where[Object.getOwnPropertySymbols(params.where)[0]]) {
              findParams.push(param)
            }
          }
          let result = null
          result = data.find((item) => {
            if (findParams.length === 1) {
              return item.tntAddr === findParams[0].tntAddr
            } else {
              return item.tntAddr === findParams[0].tntAddr || item.publicUri === findParams[1].publicUri
            }
          })
          return result
        },
        create: (params) => {
          let row = {
            tntAddr: params.tntAddr,
            publicUri: params.publicUri,
            hmacKey: hmacKey
          }
          data.push(row)
          return row
        }
      })

      app.overrideGetTNTGrainsBalanceForAddressAsync(async (addr) => { return 500000000000 })

      request(server)
        .post('/nodes')
        .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
        .send({ tnt_addr: tntAddr1, public_uri: publicUri1 })
        .expect(200)
        .end((err, res) => {
          expect(err).to.equal(null)

          request(server)
            .post('/nodes')
            .set('X-Node-Version', process.env.MIN_NODE_VERSION_NEW)
            .send({ tnt_addr: tntAddr2, public_uri: publicUri2 })
            .expect(200)
            .end((err, res) => {
              expect(err).to.equal(null)
              // HMAC-SHA256(hmac-key, TNT_ADDRESS|IP|YYYYMMDDHHMM)
              let hash = crypto.createHmac('sha256', res.body.hmac_key)
              let formattedDate = moment().utc().format('YYYYMMDDHHmm')
              let hmacTxt = [tntAddr1, publicUri2, formattedDate].join('')
              let calculatedHMAC = hash.update(hmacTxt).digest('hex')

              request(server)
                .put('/nodes/' + tntAddr1)
                .set('X-Node-Version', process.env.MIN_NODE_VERSION_EXISTING)
                .send({ public_uri: publicUri2, hmac: calculatedHMAC })
                .expect('Content-type', /json/)
                .expect(409)
                .end((err, res) => {
                  expect(err).to.equal(null)
                  expect(res.body).to.have.property('code')
                    .and.to.be.a('string')
                    .and.to.equal('ConflictError')
                  expect(res.body).to.have.property('message')
                    .and.to.be.a('string')
                    .and.to.equal('the public URI provided is already registered')
                  done()
                })
            })
        })
    })
  })
})

describe('Functions', () => {
  describe('calling generatePostHashResponse with one hash', () => {
    it('should return proper response object', (done) => {
      let res = hashes.generatePostHashResponse('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12', { tntCredit: 9999 })
      expect(res).to.have.property('hash_id')
      expect(res).to.have.property('hash').and.to.equal('ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12ab12')
      expect(res).to.have.property('submitted_at')
      expect(res).to.have.property('processing_hints')
      expect(res.processing_hints).to.have.property('cal').and.to.be.a('string')
      expect(res.processing_hints).to.have.property('eth').and.to.be.a('string')
      expect(res.processing_hints).to.have.property('btc').and.to.be.a('string')
      expect(res).to.have.property('tnt_credit_balance').and.to.equal(9999)
      done()
    })
  })
})
