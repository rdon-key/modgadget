# Public embedded MGF font assets

The MGF binaries in this directory are generated assets. Do not edit the
binary files by hand.

- Source repository: `modgadget-font-assets`
- Source repository commit: `5de8b7ffa46ab97e7dac658ae4cffa670480affb`
- Generator repository commit: `36141f000687df39c3dad106f5455f55be64e6b0`

| Font | Source path | Embedded output path | SHA-256 |
| --- | --- | --- | --- |
| Shinonome 12 | `fonts/shinonome12/shinonome12-full.mgf` | `font/shinonome12/shinonome12-full.mgf` | `3b2ee24462103e1bccbb4fbb1fcc943c61eb74b66dbae7120ff2463d74a0f136` |
| Spleen 8x16 | `fonts/spleen-8x16/spleen-8x16-full.mgf` | `font/spleen8x16/spleen-8x16-full.mgf` | `0b78fcb25a1096801e4b90b0415f7bf8ad87b015ff9f9ef8124be7177139d3ba` |

See `MGF-ASSETS.md` in the `modgadget-font-assets` repository for the exact
regeneration commands and source BDF provenance.

## Efont Biwidth assets

The Efont Biwidth assets below were imported from source repository commit
`66b23862fe6ec38ef362ef74596912231ec14a51` and generated with generator
repository commit `36141f000687df39c3dad106f5455f55be64e6b0`.

| Font | Source BDF | Source MGF | Embedded output | BDF SHA-256 | MGF SHA-256 | FileSize |
| --- | --- | --- | --- | --- | --- | ---: |
| Efont Biwidth 16 | `fonts/efont16/b16.bdf` | `fonts/efont16/efont16-full.mgf` | `font/efont16/efont16-full.mgf` | `2dd69898adba95a4bb47a7490b54ccb0fc95bd59007fc63fe2c6bb29a9bc5cb5` | `0cbbcc0b0a3845be11d5cd958c2ea092afa6fdd82be9ae82f6d1a87274e9ea16` | 1167336 |
| Efont Biwidth 24 | `fonts/efont24/b24.bdf` | `fonts/efont24/efont24-full.mgf` | `font/efont24/efont24-full.mgf` | `f03ad7d046b2b7e976bfba89f500117ef8d11c370055ba4adede866023754ad6` | `d87645e7b45cbf9e9758349a9a337bd38ef832d781e96c19d0596f113ca8f4a7` | 2706726 |

The corresponding upstream notices are preserved under
`LICENSES/efont-unicode-bdf-0.4.2/`. Its byte-identical `COPYRIGHT` and
`README.shinonome` files already exist as `LICENSES/shinonome-BSD-3-Clause.txt`
and `LICENSES/shinonome-NOTICE.txt`, so they are not duplicated. Do not edit
the MGF binaries by hand; refer to `MGF-ASSETS.md` in the source repository
when regenerating them.
