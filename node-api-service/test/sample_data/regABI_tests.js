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
    inputs: [
      {
        name: '',
        type: 'address'
      }
    ],
    name: 'nodes',
    outputs: [
      {
        name: 'nodeIp',
        type: 'uint32'
      },
      {
        name: 'rewardsAddr',
        type: 'address'
      },
      {
        name: 'isStaked',
        type: 'bool'
      },
      {
        name: 'amountStaked',
        type: 'uint256'
      },
      {
        name: 'stakeLockedUntil',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x189a5a17'
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
    inputs: [
      {
        name: '',
        type: 'uint32'
      }
    ],
    name: 'allocatedIps',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x4b0624bb'
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
    constant: true,
    inputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    name: 'whitelistedCoresArr',
    outputs: [
      {
        name: '',
        type: 'address'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x79a382fe'
  },
  {
    constant: true,
    inputs: [],
    name: 'CORE_STAKING_DURATION',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x7c8f3840'
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
    inputs: [
      {
        name: '',
        type: 'address'
      }
    ],
    name: 'cores',
    outputs: [
      {
        name: 'coreIp',
        type: 'uint32'
      },
      {
        name: 'isStaked',
        type: 'bool'
      },
      {
        name: 'isHealthy',
        type: 'bool'
      },
      {
        name: 'amountStaked',
        type: 'uint256'
      },
      {
        name: 'stakeLockedUntil',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x85ad32c4'
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
    name: 'NODE_STAKING_AMOUNT',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0xcfc635d4'
  },
  {
    constant: true,
    inputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    name: 'coresArr',
    outputs: [
      {
        name: '',
        type: 'address'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0xd2c27afc'
  },
  {
    constant: true,
    inputs: [],
    name: 'CORE_STAKING_AMOUNT',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0xe5d0bf7d'
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
    constant: true,
    inputs: [],
    name: 'NODE_STAKING_DURATION',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0xf353e749'
  },
  {
    inputs: [
      {
        name: '_token',
        type: 'address'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'constructor',
    signature: 'constructor'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        name: '_sender',
        type: 'address'
      },
      {
        indexed: false,
        name: '_nodeIp',
        type: 'uint32'
      },
      {
        indexed: false,
        name: '_rewardsAddr',
        type: 'address'
      },
      {
        indexed: false,
        name: '_amountStaked',
        type: 'uint256'
      },
      {
        indexed: false,
        name: '_duration',
        type: 'uint256'
      }
    ],
    name: 'NodeStaked',
    type: 'event',
    signature: '0x3a506917cfa3b7b4b782b6385f02923905fd60e81cc7b673e3a1453f834af259'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        name: '_sender',
        type: 'address'
      },
      {
        indexed: false,
        name: '_nodeIp',
        type: 'uint32'
      },
      {
        indexed: false,
        name: '_amountStaked',
        type: 'uint256'
      },
      {
        indexed: false,
        name: '_duration',
        type: 'uint256'
      }
    ],
    name: 'NodeStakeUpdated',
    type: 'event',
    signature: '0x32ea73b36321c1e1a1ab6d4c06e53a3118dceb49c9001e0af83f50764cdcd430'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        name: '_sender',
        type: 'address'
      },
      {
        indexed: false,
        name: '_nodeIp',
        type: 'uint32'
      },
      {
        indexed: false,
        name: '_amountStaked',
        type: 'uint256'
      }
    ],
    name: 'NodeUnStaked',
    type: 'event',
    signature: '0x7723f01f1f38e47e511a270774d4dcc884a882c51ec5b002a5a94995afcd1521'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        name: '_sender',
        type: 'address'
      },
      {
        indexed: false,
        name: '_coreIp',
        type: 'uint32'
      },
      {
        indexed: false,
        name: '_isHealthy',
        type: 'bool'
      },
      {
        indexed: false,
        name: '_amountStaked',
        type: 'uint256'
      },
      {
        indexed: false,
        name: '_duration',
        type: 'uint256'
      }
    ],
    name: 'CoreStaked',
    type: 'event',
    signature: '0x796a2d9218734911a0d7003fbefb8d548e07e43561ca84c46eebbecb27360b77'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        name: '_sender',
        type: 'address'
      },
      {
        indexed: false,
        name: '_coreIp',
        type: 'uint32'
      },
      {
        indexed: false,
        name: '_isHealthy',
        type: 'bool'
      },
      {
        indexed: false,
        name: '_amountStaked',
        type: 'uint256'
      },
      {
        indexed: false,
        name: '_duration',
        type: 'uint256'
      }
    ],
    name: 'CoreStakeUpdated',
    type: 'event',
    signature: '0x72da1794d27e553658e2fa30770730ecdbf2f0b8dfd8540094ca9bc75d8ae0f6'
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        name: '_sender',
        type: 'address'
      },
      {
        indexed: false,
        name: '_coreIp',
        type: 'uint32'
      },
      {
        indexed: false,
        name: '_amountStaked',
        type: 'uint256'
      }
    ],
    name: 'CoreUnStaked',
    type: 'event',
    signature: '0x341d92626b7f883cbb4fd3b9bc7daed691a5559249e8a16fee8b1bb0d0be71f3'
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
    constant: false,
    inputs: [
      {
        name: '_nodeIp',
        type: 'uint32'
      },
      {
        name: '_rewardsAddr',
        type: 'address'
      }
    ],
    name: 'stake',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0xeff49c8b'
  },
  {
    constant: false,
    inputs: [
      {
        name: '_coreIp',
        type: 'uint32'
      }
    ],
    name: 'stakeCore',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0xecb9c311'
  },
  {
    constant: false,
    inputs: [
      {
        name: '_nodeIp',
        type: 'uint32'
      }
    ],
    name: 'updateStake',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0x8689baf2'
  },
  {
    constant: false,
    inputs: [
      {
        name: '_coreIp',
        type: 'uint32'
      }
    ],
    name: 'updateStakeCore',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0xd2f05a51'
  },
  {
    constant: false,
    inputs: [],
    name: 'unStake',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0x73cf575a'
  },
  {
    constant: false,
    inputs: [],
    name: 'unStakeCore',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0xdc964893'
  },
  {
    constant: true,
    inputs: [
      {
        name: 'addr',
        type: 'address'
      }
    ],
    name: 'totalStakedFor',
    outputs: [
      {
        name: 'amount',
        type: 'uint256'
      },
      {
        name: 'unlocks_at',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x4b341aed'
  },
  {
    constant: true,
    inputs: [],
    name: 'getCoreCount',
    outputs: [
      {
        name: '',
        type: 'uint256'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0xf1730874'
  },
  {
    constant: true,
    inputs: [
      {
        name: '_address',
        type: 'address'
      }
    ],
    name: 'isHealthyCore',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'view',
    type: 'function',
    signature: '0x9273cce8'
  },
  {
    constant: false,
    inputs: [
      {
        name: '_address',
        type: 'address'
      }
    ],
    name: 'whitelistCore',
    outputs: [
      {
        name: '',
        type: 'bool'
      }
    ],
    payable: false,
    stateMutability: 'nonpayable',
    type: 'function',
    signature: '0xd8a8788d'
  }
]
