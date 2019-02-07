# Chainpoint Binary

[![code style: prettier](https://img.shields.io/badge/code_style-prettier-ff69b4.svg?style=flat-square)](https://github.com/prettier/prettier)
[![npm](https://img.shields.io/npm/l/chainpoint-binary.svg)](https://www.npmjs.com/package/chainpoint-binary)
[![npm](https://img.shields.io/npm/v/chainpoint-binary.svg)](https://www.npmjs.com/package/chainpoint-binary)

A Javascript library for serializing/deserializing a [Chainpoint v3 proof](https://chainpoint.org/) between its Javascript Object/JSON and compressed binary forms.

## About the Chainpoint Binary Format

[Chainpoint](https://chainpoint.org/) v3 proofs are found in one of two forms; a Javascript Object, or its JSON String form, or a compressed binary. The binary form is designed to be:

- easy to use
- significantly smaller than JSON text
- simple and safe to convert to and from
- lossless across serialization/deserialization
- usable cross platform
- usable across many development languages
- easy to parse and understand
- lightweight

To acheive these goals we have chosen a simple mechanism for serializing proofs to binary form using two very common and well supported tools, [MessagePack](http://msgpack.org/index.html) and [zlib](http://zlib.net/) deflate:

### MessagePack

> MessagePack is an efficient binary serialization format. It lets you exchange data among multiple languages like JSON. But it's faster and smaller. Small integers are encoded into a single byte, and typical short strings require only one extra byte in addition to the strings themselves.

MessagePack is currently supported by over [50 different programming languages](http://msgpack.org/index.html#languages).

This library makes use of the `msgpack-lite` implementation for Node.js. You can [try it out](http://kawanet.github.io/msgpack-lite/) or read the [documentation](https://github.com/kawanet/msgpack-lite).

### zlib

MessagePack encoded proofs are then compressed with the `zlib` deflate compression function.

> zlib is designed to be a free, general-purpose, legally unencumbered -- that is, not covered by any patents -- lossless data-compression library for use on virtually any computer hardware and operating system. The zlib data format is itself portable across platforms.

This library makes use of the `pako` implementation of `zlib` which can be [found here](https://github.com/nodeca/pako).

## Chainpoint File Extensions

Chainpoint proofs should typically have a filename that is either the same as the `hash_id`
that was returned when the `hash` was submitted to Chainpoint or the same filename as the original file that was hashed. A `hash_id` is a [Version 1 UUID](https://en.wikipedia.org/wiki/Universally_unique_identifier#Version_1_.28date-time_and_MAC_address.29) and is used as a handle to retrieve a proof.

When storing a proof alongside an original file on a filesystem that was hashed and sent to Chainpoint it may be more convenient to mirror the original filename and add the appropriate file extension.

Chainpoint v3 proofs should have one of the following file name + file extensions:

### `<filename>.chp.json`

Uncompressed JSON string proof mirroring the original filename

### `<filename>.chp`

MessagePack + zlib binary proof mirroring the original filename

### `<hash_id>.chp.json`

Uncompressed JSON string proof named after the `hash_id`

### `<hash_id>.chp`

MessagePack + zlib binary proof named after the `hash_id`

## Installation

```
$ npm install --save chainpoint-binary
```

or

```
yarn add chainpoint-binary
```

## Try it out

Try out the conversion of a sample proof, or one of your own, to Binary and back to JSON. This example code can be found in the `docs` directory.

https://chainpoint.org/chainpoint-binary/

## Usage

### Node.js

#### `objectToBinary`

#### `objectToBinarySync`

This function converts a Chainpoint proof in JSON String or Javascript Object form to a Buffer containing the standard binary form. The incoming Object will be validated against
the formal [Chainpoint Proof JSON Schema](https://github.com/chainpoint/chainpoint-proof-json-schema).

```js
const cpb = require('chainpoint-binary')

// Valid proof in JSON or JS Object form
let chainpointProofObject = {...}

cpb.objectToBinary(chainpointProofObject, function (err, proofBinary) {
    if (err) {
      // if an error occurs, the error message will return here
    } else {
      // proofBinary is a Buffer representing the binary form of a Chainpoint proof
    }
})
```

```js
const cpb = require('chainpoint-binary')

// Valid proof in JSON or JS Object form
let chainpointProofObject = {...}

let proofBinary = cpb.objectToBinarySync(chainpointProofObject)
```

#### `objectToBase64`

#### `objectToBase64Sync`

This function converts a Chainpoint proof in JSON String or Javascript Object form to a Base64 encoded string of the standard binary form. The incoming Object will be validated against
the formal [Chainpoint Proof JSON Schema](https://github.com/chainpoint/chainpoint-proof-json-schema).

```js
const cpb = require('chainpoint-binary')

// Valid proof in JSON or JS Object form
let chainpointProofObject = {...}

cpb.objectToBase64(chainpointProofObject, function (err, proofBase64) {
    if (err) {
      // if an error occurs, the error message will return here
    } else {
      // proofBase64 is a Base64 encoded string representing the binary form of a Chainpoint proof
    }
})
```

```js
const cpb = require('chainpoint-binary')

// Valid proof in JSON or JS Object form
let chainpointProofObject = {...}

let proofBase64 = cpb.objectToBase64Sync(chainpointProofObject)
```

#### `binaryToObject`

#### `binaryToObjectSync`

This function converts a Chainpoint binary proof to a Javascript Object. A Hexadecimal string or Base64 string in place of a Buffer is also acceptable as input. The outgoing Object will be validated against the formal [Chainpoint Proof JSON Schema](https://github.com/chainpoint/chainpoint-proof-json-schema) before being returned.

```js
const cpb = require('chainpoint-binary')

// Valid proof in Buffer, Hex String, or Base64 String form
let chainpointProofBinaryBuffer = [...]

cpb.binaryToObject(chainpointProofBinaryBuffer, function (err, proofObject) {
    if (err) {
      // if an error occurs, the error message will return here
    } else {
      // proofObject contains the Javascript Object form of a Chainpoint proof
      // Wrap this with JSON.stringify() for the JSON form as needed.
    }
})
```

```js
const cpb = require('chainpoint-binary')

// Valid proof in Buffer, Hex String, or Base64 String form
let chainpointProofBinaryBuffer = [...]

let proofObject = cpb.binaryToObjectSync(chainpointProofBinaryBuffer)
```

### Browser

Note : You can copy `docs/bundle.js` into your app to include in a script tag, or use a CDN to serve it to your app directly.

Or install the npm package in a place available to your web pages and set the script `src` tag to something like the following. A set of window global functions `chainpointBinary.binaryToObject()` and `chainpointBinary.objectToBinary()` will be available and operate the same as the Node.js example above.

```
<script src="./node_modules/chainpoint-binary/docs/bundle.js">
```

## Development

### Code Formatting & Linting

This project makes use of [Prettier](https://prettier.io/) & [ESLint](https://eslint.org/) to maintain clean, and consistently styled, code.

You can run ESLint manually using the CLI:

```sh
./node_modules/.bin/eslint .
```

You can run Prettier manually to see which files would be re-formatted by it. For example:

```sh
./node_modules/.bin/prettier -l *.js
```

This project is coded in the [Visual Studio Code](https://code.visualstudio.com/) IDE and we use the following plugins to auto-format and report on linting issues during development:

[vscode-eslint](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint)

[EditorConfig](https://marketplace.visualstudio.com/items?itemName=EditorConfig.EditorConfig)
