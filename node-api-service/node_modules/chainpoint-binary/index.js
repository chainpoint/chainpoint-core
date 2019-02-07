/* Copyright 2017 Tierion
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*     http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/

const chpSchema = require('chainpoint-proof-json-schema')
const mpack = require('msgpack-lite')
const pako = require('pako')

let isValidHex = function(hex) {
  var hexRegex = /^[0-9A-Fa-f]{2,}$/
  var hasHexChars = hexRegex.test(hex)
  var hasEvenLen = hex.length % 2 === 0

  if (hasHexChars && hasEvenLen) return true
  return false
}

let objectToBinary = (proofObj, cb) => {
  if (!proofObj) return cb('No proof Object or JSON string arg provided')

  // Handle a JSON String arg
  if (typeof proofObj === 'string') {
    try {
      proofObj = JSON.parse(proofObj)
    } catch (err) {
      return cb('Invalid JSON string proof provided')
    }
  }

  // A well-formed, schema compliant Chainpoint proof?
  let validateResult = chpSchema.validate(proofObj)
  if (!validateResult.valid) return cb('Chainpoint v3 schema validation error')

  let deflatedProof = pako.deflate(mpack.encode(proofObj))
  return cb(null, Buffer.from(deflatedProof))
}

let objectToBinarySync = proofObj => {
  if (!proofObj) throw new Error('No proof Object or JSON string arg provided')

  // Handle a JSON String arg
  if (typeof proofObj === 'string') {
    try {
      proofObj = JSON.parse(proofObj)
    } catch (err) {
      throw new Error('Invalid JSON string proof provided')
    }
  }

  // A well-formed, schema compliant Chainpoint proof?
  let validateResult = chpSchema.validate(proofObj)
  if (!validateResult.valid)
    throw new Error('Chainpoint v3 schema validation error')

  let deflatedProof = pako.deflate(mpack.encode(proofObj))
  return Buffer.from(deflatedProof)
}

let objectToBase64 = (proofObj, cb) => {
  objectToBinary(proofObj, (err, proofBinary) => {
    if (err) return cb(err)
    return cb(null, proofBinary.toString('base64'))
  })
}

let objectToBase64Sync = proofObj => {
  let proofBinary = objectToBinarySync(proofObj)
  return proofBinary.toString('base64')
}

let binaryToObject = (proof, cb) => {
  if (!proof) return cb('No binary proof arg provided')

  try {
    // Handle a Hexadecimal String arg in addition to a Buffer
    if (!Buffer.isBuffer(proof)) {
      if (isValidHex(proof)) {
        proof = Buffer.from(proof, 'hex')
      } else {
        proof = Buffer.from(proof, 'base64')
      }
    }

    let unpackedProof = mpack.decode(pako.inflate(proof))
    if (!chpSchema.validate(unpackedProof).valid)
      return cb('Chainpoint v3 schema validation error')
    return cb(null, unpackedProof)
  } catch (e) {
    return cb('Could not parse Chainpoint v3 binary')
  }
}

let binaryToObjectSync = proof => {
  if (!proof) throw new Error('No binary proof arg provided')

  try {
    // Handle a Hexadecimal String arg in addition to a Buffer
    if (!Buffer.isBuffer(proof)) {
      if (isValidHex(proof)) {
        proof = Buffer.from(proof, 'hex')
      } else {
        proof = Buffer.from(proof, 'base64')
      }
    }

    let unpackedProof = mpack.decode(pako.inflate(proof))
    if (!chpSchema.validate(unpackedProof).valid)
      throw new Error('Chainpoint v3 schema validation error')
    return unpackedProof
  } catch (e) {
    throw new Error('Could not parse Chainpoint v3 binary')
  }
}

module.exports = {
  objectToBinary: objectToBinary,
  objectToBase64: objectToBase64,
  binaryToObject: binaryToObject,
  objectToBinarySync: objectToBinarySync,
  objectToBase64Sync: objectToBase64Sync,
  binaryToObjectSync: binaryToObjectSync
}
