module.exports = [
  {
    constant: true,
    inputs: [],
    name: 'name',
    outputs: [
      {
        name: '',
        type: 'string'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x06fdde03'
  },
  {
    constant: true,
    inputs: [],
    name: 'totalSupply',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x18160ddd'
  },
  {
    constant: true,
    inputs: [],
    name: 'INITIAL_SUPPLY',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x2ff2e9dc'
  },
  {
    constant: true,
    inputs: [],
    name: 'decimals',
    outputs: [
      {
        name: '',
        type: 'uint8'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x313ce567'
  },
  {
    constant: false,
    inputs: [],
    name: 'unpause',
    outputs: [],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0x3f4ba83a'
  },
  {
    constant: true,
    inputs: [
      {
        name: 'account',
        type: 'address'
      }
    ],
    name: 'isPauser',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x46fbf68e'
  },
  {
    constant: true,
    inputs: [],
    name: 'coreLastMintedAtBlock',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x4c5f25dd'
  },
  {
    constant: true,
    inputs: [],
    name: 'mintAmount',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x5a2bcc18'
  },
  {
    constant: true,
    inputs: [],
    name: 'nodeLastMintedAtBlock',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x5a8ef28b'
  },
  {
    constant: true,
    inputs: [
      {
        name: '',
        type: 'address'
      },
      {
        name: '',
        type: 'address'
      }
    ],
    name: 'allowed',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x5c658165'
  },
  {
    constant: true,
    inputs: [],
    name: 'paused',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x5c975abb'
  },
  {
    constant: false,
    inputs: [],
    name: 'renouncePauser',
    outputs: [],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0x6ef8d66d'
  },
  {
    constant: true,
    inputs: [
      {
        name: '_owner',
        type: 'address'
      }
    ],
    name: 'balanceOf',
    outputs: [
      {
        name: 'balance',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x70a08231'
  },
  {
    constant: false,
    inputs: [],
    name: 'renounceOwnership',
    outputs: [],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0x715018a6'
  },
  {
    constant: false,
    inputs: [
      {
        name: 'account',
        type: 'address'
      }
    ],
    name: 'addPauser',
    outputs: [],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0x82dc1ec4'
  },
  {
    constant: false,
    inputs: [],
    name: 'pause',
    outputs: [],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0x8456cb59'
  },
  {
    constant: true,
    inputs: [],
    name: 'owner',
    outputs: [
      {
        name: '',
        type: 'address'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x8da5cb5b'
  },
  {
    constant: true,
    inputs: [],
    name: 'isOwner',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x8f32d59b'
  },
  {
    constant: true,
    inputs: [],
    name: 'symbol',
    outputs: [
      {
        name: '',
        type: 'string'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x95d89b41'
  },
  {
    constant: true,
    inputs: [],
    name: 'mintingInterval',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0xa3059ef0'
  },
  {
    constant: true,
    inputs: [
      {
        name: '_owner',
        type: 'address'
      },
      {
        name: '_spender',
        type: 'address'
      }
    ],
    name: 'allowance',
    outputs: [
      {
        name: 'remaining',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0xdd62ed3e'
  },
  {
    constant: false,
    inputs: [
      {
        name: 'newOwner',
        type: 'address'
      }
    ],
    name: 'transferOwnership',
    outputs: [],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0xf2fde38b'
  },
  {
    inputs: [],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'constructor',
    signature: 'constructor'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        name: '_nodes',
        type: 'address[]'
      },
      {
        indexed: false,
        name: '_mintAmount',
        type: 'uint256'
      },
      {
        indexed: false,
        name: '_blockHeight',
        type: 'uint256'
      }
    ],
    name: 'Mint',
    type: 'event',
    signature: '0x35cc581d6103df194c8c20f437d4b065c9107097eb2ba6f9702489434e03b64c'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        name: '_buyer',
        type: 'address'
      },
      {
        indexed: false,
        name: '_value',
        type: 'uint256'
      }
    ],
    name: 'UsagePurchased',
    type: 'event',
    signature: '0x0651f693d262f8b7c2f4a40577914247c33a4eb3fa57d735c8ac49be550021cb'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        name: 'account',
        type: 'address'
      }
    ],
    name: 'Paused',
    type: 'event',
    signature: '0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: false,
        name: 'account',
        type: 'address'
      }
    ],
    name: 'Unpaused',
    type: 'event',
    signature: '0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        name: 'account',
        type: 'address'
      }
    ],
    name: 'PauserAdded',
    type: 'event',
    signature: '0x6719d08c1888103bea251a4ed56406bd0c3e69723c8a1686e017e7bbe159b6f8'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        name: 'account',
        type: 'address'
      }
    ],
    name: 'PauserRemoved',
    type: 'event',
    signature: '0xcd265ebaf09df2871cc7bd4133404a235ba12eff2041bb89d9c714a2621c7c7e'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        name: 'previousOwner',
        type: 'address'
      },
      {
        indexed: true,
        name: 'newOwner',
        type: 'address'
      }
    ],
    name: 'OwnershipTransferred',
    type: 'event',
    signature: '0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        name: 'owner',
        type: 'address'
      },
      {
        indexed: true,
        name: 'spender',
        type: 'address'
      },
      {
        indexed: false,
        name: 'value',
        type: 'uint256'
      }
    ],
    name: 'Approval',
    type: 'event',
    signature: '0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        name: 'from',
        type: 'address'
      },
      {
        indexed: true,
        name: 'to',
        type: 'address'
      },
      {
        indexed: false,
        name: 'value',
        type: 'uint256'
      }
    ],
    name: 'Transfer',
    type: 'event',
    signature: '0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef'
  },
  {
    constant: false,
    inputs: [
      {
        name: '_to',
        type: 'address'
      },
      {
        name: '_value',
        type: 'uint256'
      }
    ],
    name: 'transfer',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0xa9059cbb'
  },
  {
    constant: false,
    inputs: [
      {
        name: '_from',
        type: 'address'
      },
      {
        name: '_to',
        type: 'address'
      },
      {
        name: '_value',
        type: 'uint256'
      }
    ],
    name: 'transferFrom',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0x23b872dd'
  },
  {
    constant: false,
    inputs: [
      {
        name: '_spender',
        type: 'address'
      },
      {
        name: '_value',
        type: 'uint256'
      }
    ],
    name: 'approve',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0x095ea7b3'
  },
  {
    constant: false,
    inputs: [
      {
        name: '_nodes',
        type: 'address[]'
      },
      {
        name: '_hash',
        type: 'bytes32'
      },
      {
        name: 'signature1',
        type: 'bytes'
      },
      {
        name: 'signature2',
        type: 'bytes'
      },
      {
        name: 'signature3',
        type: 'bytes'
      },
      {
        name: 'signature4',
        type: 'bytes'
      },
      {
        name: 'signature5',
        type: 'bytes'
      },
      {
        name: 'signature6',
        type: 'bytes'
      }
    ],
    name: 'mint',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0x382e6fdb'
  },
  {
    constant: false,
    inputs: [
      {
        name: '_value',
        type: 'uint256'
      }
    ],
    name: 'purchaseUsage',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0xb55968ec'
  },
  {
    constant: false,
    inputs: [
      {
        name: '_addr',
        type: 'address'
      }
    ],
    name: 'setChainpointRegistry',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0x44ea8d44'
  },
  {
    constant: true,
    inputs: [
      {
        name: 'hash',
        type: 'bytes32'
      },
      {
        name: 'signature',
        type: 'bytes'
      }
    ],
    name: 'recover',
    outputs: [
      {
        name: '',
        type: 'address'
      }
    ],
    payable: false,
    stateMutability: 'pure',
    type: 'function',
    signature: '0x19045a25'
  },
  {
    constant: true,
    inputs: [
      {
        name: 'hash',
        type: 'bytes32'
      }
    ],
    name: 'toEthSignedMessageHash',
    outputs: [
      {
        name: '',
        type: 'bytes32'
      }
    ],
    payable: false,
    stateMutability: 'pure',
    type: 'function',
    signature: '0x918a15cf'
  }
]
