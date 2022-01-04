/*
MIT License

Copyright (c) 2021 Prysmatic Labs

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

This code is based on Intel's implementation found in
	https://github.com/intel/intel-ipsec-mb
Copied parts are
	Copyright (c) 2012-2021, Intel Corporation
*/

#ifdef __x86_64__
.intel_syntax noprefix
.section .rodata
.align 64
.LK256_8:
	.quad	0x428a2f98428a2f98, 0x428a2f98428a2f98
	.quad	0x428a2f98428a2f98, 0x428a2f98428a2f98
	.quad	0x7137449171374491, 0x7137449171374491
	.quad	0x7137449171374491, 0x7137449171374491
	.quad	0xb5c0fbcfb5c0fbcf, 0xb5c0fbcfb5c0fbcf
	.quad	0xb5c0fbcfb5c0fbcf, 0xb5c0fbcfb5c0fbcf
	.quad	0xe9b5dba5e9b5dba5, 0xe9b5dba5e9b5dba5
	.quad	0xe9b5dba5e9b5dba5, 0xe9b5dba5e9b5dba5
	.quad	0x3956c25b3956c25b, 0x3956c25b3956c25b
	.quad	0x3956c25b3956c25b, 0x3956c25b3956c25b
	.quad	0x59f111f159f111f1, 0x59f111f159f111f1
	.quad	0x59f111f159f111f1, 0x59f111f159f111f1
	.quad	0x923f82a4923f82a4, 0x923f82a4923f82a4
	.quad	0x923f82a4923f82a4, 0x923f82a4923f82a4
	.quad	0xab1c5ed5ab1c5ed5, 0xab1c5ed5ab1c5ed5
	.quad	0xab1c5ed5ab1c5ed5, 0xab1c5ed5ab1c5ed5
	.quad	0xd807aa98d807aa98, 0xd807aa98d807aa98
	.quad	0xd807aa98d807aa98, 0xd807aa98d807aa98
	.quad	0x12835b0112835b01, 0x12835b0112835b01
	.quad	0x12835b0112835b01, 0x12835b0112835b01
	.quad	0x243185be243185be, 0x243185be243185be
	.quad	0x243185be243185be, 0x243185be243185be
	.quad	0x550c7dc3550c7dc3, 0x550c7dc3550c7dc3
	.quad	0x550c7dc3550c7dc3, 0x550c7dc3550c7dc3
	.quad	0x72be5d7472be5d74, 0x72be5d7472be5d74
	.quad	0x72be5d7472be5d74, 0x72be5d7472be5d74
	.quad	0x80deb1fe80deb1fe, 0x80deb1fe80deb1fe
	.quad	0x80deb1fe80deb1fe, 0x80deb1fe80deb1fe
	.quad	0x9bdc06a79bdc06a7, 0x9bdc06a79bdc06a7
	.quad	0x9bdc06a79bdc06a7, 0x9bdc06a79bdc06a7
	.quad	0xc19bf174c19bf174, 0xc19bf174c19bf174
	.quad	0xc19bf174c19bf174, 0xc19bf174c19bf174
	.quad	0xe49b69c1e49b69c1, 0xe49b69c1e49b69c1
	.quad	0xe49b69c1e49b69c1, 0xe49b69c1e49b69c1
	.quad	0xefbe4786efbe4786, 0xefbe4786efbe4786
	.quad	0xefbe4786efbe4786, 0xefbe4786efbe4786
	.quad	0x0fc19dc60fc19dc6, 0x0fc19dc60fc19dc6
	.quad	0x0fc19dc60fc19dc6, 0x0fc19dc60fc19dc6
	.quad	0x240ca1cc240ca1cc, 0x240ca1cc240ca1cc
	.quad	0x240ca1cc240ca1cc, 0x240ca1cc240ca1cc
	.quad	0x2de92c6f2de92c6f, 0x2de92c6f2de92c6f
	.quad	0x2de92c6f2de92c6f, 0x2de92c6f2de92c6f
	.quad	0x4a7484aa4a7484aa, 0x4a7484aa4a7484aa
	.quad	0x4a7484aa4a7484aa, 0x4a7484aa4a7484aa
	.quad	0x5cb0a9dc5cb0a9dc, 0x5cb0a9dc5cb0a9dc
	.quad	0x5cb0a9dc5cb0a9dc, 0x5cb0a9dc5cb0a9dc
	.quad	0x76f988da76f988da, 0x76f988da76f988da
	.quad	0x76f988da76f988da, 0x76f988da76f988da
	.quad	0x983e5152983e5152, 0x983e5152983e5152
	.quad	0x983e5152983e5152, 0x983e5152983e5152
	.quad	0xa831c66da831c66d, 0xa831c66da831c66d
	.quad	0xa831c66da831c66d, 0xa831c66da831c66d
	.quad	0xb00327c8b00327c8, 0xb00327c8b00327c8
	.quad	0xb00327c8b00327c8, 0xb00327c8b00327c8
	.quad	0xbf597fc7bf597fc7, 0xbf597fc7bf597fc7
	.quad	0xbf597fc7bf597fc7, 0xbf597fc7bf597fc7
	.quad	0xc6e00bf3c6e00bf3, 0xc6e00bf3c6e00bf3
	.quad	0xc6e00bf3c6e00bf3, 0xc6e00bf3c6e00bf3
	.quad	0xd5a79147d5a79147, 0xd5a79147d5a79147
	.quad	0xd5a79147d5a79147, 0xd5a79147d5a79147
	.quad	0x06ca635106ca6351, 0x06ca635106ca6351
	.quad	0x06ca635106ca6351, 0x06ca635106ca6351
	.quad	0x1429296714292967, 0x1429296714292967
	.quad	0x1429296714292967, 0x1429296714292967
	.quad	0x27b70a8527b70a85, 0x27b70a8527b70a85
	.quad	0x27b70a8527b70a85, 0x27b70a8527b70a85
	.quad	0x2e1b21382e1b2138, 0x2e1b21382e1b2138
	.quad	0x2e1b21382e1b2138, 0x2e1b21382e1b2138
	.quad	0x4d2c6dfc4d2c6dfc, 0x4d2c6dfc4d2c6dfc
	.quad	0x4d2c6dfc4d2c6dfc, 0x4d2c6dfc4d2c6dfc
	.quad	0x53380d1353380d13, 0x53380d1353380d13
	.quad	0x53380d1353380d13, 0x53380d1353380d13
	.quad	0x650a7354650a7354, 0x650a7354650a7354
	.quad	0x650a7354650a7354, 0x650a7354650a7354
	.quad	0x766a0abb766a0abb, 0x766a0abb766a0abb
	.quad	0x766a0abb766a0abb, 0x766a0abb766a0abb
	.quad	0x81c2c92e81c2c92e, 0x81c2c92e81c2c92e
	.quad	0x81c2c92e81c2c92e, 0x81c2c92e81c2c92e
	.quad	0x92722c8592722c85, 0x92722c8592722c85
	.quad	0x92722c8592722c85, 0x92722c8592722c85
	.quad	0xa2bfe8a1a2bfe8a1, 0xa2bfe8a1a2bfe8a1
	.quad	0xa2bfe8a1a2bfe8a1, 0xa2bfe8a1a2bfe8a1
	.quad	0xa81a664ba81a664b, 0xa81a664ba81a664b
	.quad	0xa81a664ba81a664b, 0xa81a664ba81a664b
	.quad	0xc24b8b70c24b8b70, 0xc24b8b70c24b8b70
	.quad	0xc24b8b70c24b8b70, 0xc24b8b70c24b8b70
	.quad	0xc76c51a3c76c51a3, 0xc76c51a3c76c51a3
	.quad	0xc76c51a3c76c51a3, 0xc76c51a3c76c51a3
	.quad	0xd192e819d192e819, 0xd192e819d192e819
	.quad	0xd192e819d192e819, 0xd192e819d192e819
	.quad	0xd6990624d6990624, 0xd6990624d6990624
	.quad	0xd6990624d6990624, 0xd6990624d6990624
	.quad	0xf40e3585f40e3585, 0xf40e3585f40e3585
	.quad	0xf40e3585f40e3585, 0xf40e3585f40e3585
	.quad	0x106aa070106aa070, 0x106aa070106aa070
	.quad	0x106aa070106aa070, 0x106aa070106aa070
	.quad	0x19a4c11619a4c116, 0x19a4c11619a4c116
	.quad	0x19a4c11619a4c116, 0x19a4c11619a4c116
	.quad	0x1e376c081e376c08, 0x1e376c081e376c08
	.quad	0x1e376c081e376c08, 0x1e376c081e376c08
	.quad	0x2748774c2748774c, 0x2748774c2748774c
	.quad	0x2748774c2748774c, 0x2748774c2748774c
	.quad	0x34b0bcb534b0bcb5, 0x34b0bcb534b0bcb5
	.quad	0x34b0bcb534b0bcb5, 0x34b0bcb534b0bcb5
	.quad	0x391c0cb3391c0cb3, 0x391c0cb3391c0cb3
	.quad	0x391c0cb3391c0cb3, 0x391c0cb3391c0cb3
	.quad	0x4ed8aa4a4ed8aa4a, 0x4ed8aa4a4ed8aa4a
	.quad	0x4ed8aa4a4ed8aa4a, 0x4ed8aa4a4ed8aa4a
	.quad	0x5b9cca4f5b9cca4f, 0x5b9cca4f5b9cca4f
	.quad	0x5b9cca4f5b9cca4f, 0x5b9cca4f5b9cca4f
	.quad	0x682e6ff3682e6ff3, 0x682e6ff3682e6ff3
	.quad	0x682e6ff3682e6ff3, 0x682e6ff3682e6ff3
	.quad	0x748f82ee748f82ee, 0x748f82ee748f82ee
	.quad	0x748f82ee748f82ee, 0x748f82ee748f82ee
	.quad	0x78a5636f78a5636f, 0x78a5636f78a5636f
	.quad	0x78a5636f78a5636f, 0x78a5636f78a5636f
	.quad	0x84c8781484c87814, 0x84c8781484c87814
	.quad	0x84c8781484c87814, 0x84c8781484c87814
	.quad	0x8cc702088cc70208, 0x8cc702088cc70208
	.quad	0x8cc702088cc70208, 0x8cc702088cc70208
	.quad	0x90befffa90befffa, 0x90befffa90befffa
	.quad	0x90befffa90befffa, 0x90befffa90befffa
	.quad	0xa4506ceba4506ceb, 0xa4506ceba4506ceb
	.quad	0xa4506ceba4506ceb, 0xa4506ceba4506ceb
	.quad	0xbef9a3f7bef9a3f7, 0xbef9a3f7bef9a3f7
	.quad	0xbef9a3f7bef9a3f7, 0xbef9a3f7bef9a3f7
	.quad	0xc67178f2c67178f2, 0xc67178f2c67178f2
	.quad	0xc67178f2c67178f2, 0xc67178f2c67178f2

.LPADDING_8:

	.octa     0xc28a2f98c28a2f98c28a2f98c28a2f98
	.octa     0xc28a2f98c28a2f98c28a2f98c28a2f98
	.octa 	  0x71374491713744917137449171374491
	.octa 	  0x71374491713744917137449171374491
	.octa 	  0xb5c0fbcfb5c0fbcfb5c0fbcfb5c0fbcf
	.octa 	  0xb5c0fbcfb5c0fbcfb5c0fbcfb5c0fbcf
	.octa 	  0xe9b5dba5e9b5dba5e9b5dba5e9b5dba5
	.octa 	  0xe9b5dba5e9b5dba5e9b5dba5e9b5dba5
        .octa     0x3956c25b3956c25b3956c25b3956c25b
        .octa     0x3956c25b3956c25b3956c25b3956c25b
        .octa     0x59f111f159f111f159f111f159f111f1
        .octa     0x59f111f159f111f159f111f159f111f1
        .octa     0x923f82a4923f82a4923f82a4923f82a4
        .octa     0x923f82a4923f82a4923f82a4923f82a4
        .octa     0xab1c5ed5ab1c5ed5ab1c5ed5ab1c5ed5
        .octa     0xab1c5ed5ab1c5ed5ab1c5ed5ab1c5ed5
        .octa     0xd807aa98d807aa98d807aa98d807aa98
        .octa     0xd807aa98d807aa98d807aa98d807aa98
        .octa     0x12835b0112835b0112835b0112835b01
        .octa     0x12835b0112835b0112835b0112835b01
        .octa     0x243185be243185be243185be243185be
        .octa     0x243185be243185be243185be243185be
        .octa     0x550c7dc3550c7dc3550c7dc3550c7dc3
        .octa     0x550c7dc3550c7dc3550c7dc3550c7dc3
        .octa     0x72be5d7472be5d7472be5d7472be5d74
        .octa     0x72be5d7472be5d7472be5d7472be5d74
        .octa     0x80deb1fe80deb1fe80deb1fe80deb1fe
        .octa     0x80deb1fe80deb1fe80deb1fe80deb1fe
        .octa     0x9bdc06a79bdc06a79bdc06a79bdc06a7
        .octa     0x9bdc06a79bdc06a79bdc06a79bdc06a7
        .octa     0xc19bf374c19bf374c19bf374c19bf374
        .octa     0xc19bf374c19bf374c19bf374c19bf374
        .octa     0x649b69c1649b69c1649b69c1649b69c1
        .octa     0x649b69c1649b69c1649b69c1649b69c1
        .octa     0xf0fe4786f0fe4786f0fe4786f0fe4786
        .octa     0xf0fe4786f0fe4786f0fe4786f0fe4786
        .octa     0x0fe1edc60fe1edc60fe1edc60fe1edc6
        .octa     0x0fe1edc60fe1edc60fe1edc60fe1edc6
        .octa     0x240cf254240cf254240cf254240cf254
        .octa     0x240cf254240cf254240cf254240cf254
        .octa     0x4fe9346f4fe9346f4fe9346f4fe9346f
        .octa     0x4fe9346f4fe9346f4fe9346f4fe9346f
        .octa     0x6cc984be6cc984be6cc984be6cc984be
        .octa     0x6cc984be6cc984be6cc984be6cc984be
        .octa     0x61b9411e61b9411e61b9411e61b9411e
        .octa     0x61b9411e61b9411e61b9411e61b9411e
        .octa     0x16f988fa16f988fa16f988fa16f988fa
        .octa     0x16f988fa16f988fa16f988fa16f988fa
        .octa     0xf2c65152f2c65152f2c65152f2c65152
        .octa     0xf2c65152f2c65152f2c65152f2c65152
        .octa     0xa88e5a6da88e5a6da88e5a6da88e5a6d
        .octa     0xa88e5a6da88e5a6da88e5a6da88e5a6d
        .octa     0xb019fc65b019fc65b019fc65b019fc65
        .octa     0xb019fc65b019fc65b019fc65b019fc65
        .octa     0xb9d99ec7b9d99ec7b9d99ec7b9d99ec7
        .octa     0xb9d99ec7b9d99ec7b9d99ec7b9d99ec7
        .octa     0x9a1231c39a1231c39a1231c39a1231c3
        .octa     0x9a1231c39a1231c39a1231c39a1231c3
        .octa     0xe70eeaa0e70eeaa0e70eeaa0e70eeaa0
        .octa     0xe70eeaa0e70eeaa0e70eeaa0e70eeaa0
        .octa     0xfdb1232bfdb1232bfdb1232bfdb1232b
        .octa     0xfdb1232bfdb1232bfdb1232bfdb1232b
        .octa     0xc7353eb0c7353eb0c7353eb0c7353eb0
        .octa     0xc7353eb0c7353eb0c7353eb0c7353eb0
        .octa     0x3069bad53069bad53069bad53069bad5
        .octa     0x3069bad53069bad53069bad53069bad5
        .octa     0xcb976d5fcb976d5fcb976d5fcb976d5f
        .octa     0xcb976d5fcb976d5fcb976d5fcb976d5f
        .octa     0x5a0f118f5a0f118f5a0f118f5a0f118f
        .octa     0x5a0f118f5a0f118f5a0f118f5a0f118f
        .octa     0xdc1eeefddc1eeefddc1eeefddc1eeefd
        .octa     0xdc1eeefddc1eeefddc1eeefddc1eeefd
        .octa     0x0a35b6890a35b6890a35b6890a35b689
        .octa     0x0a35b6890a35b6890a35b6890a35b689
        .octa     0xde0b7a04de0b7a04de0b7a04de0b7a04
        .octa     0xde0b7a04de0b7a04de0b7a04de0b7a04
        .octa     0x58f4ca9d58f4ca9d58f4ca9d58f4ca9d
        .octa     0x58f4ca9d58f4ca9d58f4ca9d58f4ca9d
        .octa     0xe15d5b16e15d5b16e15d5b16e15d5b16
        .octa     0xe15d5b16e15d5b16e15d5b16e15d5b16
        .octa     0x007f3e86007f3e86007f3e86007f3e86
        .octa     0x007f3e86007f3e86007f3e86007f3e86
        .octa     0x37088980370889803708898037088980
        .octa     0x37088980370889803708898037088980
        .octa     0xa507ea32a507ea32a507ea32a507ea32
        .octa     0xa507ea32a507ea32a507ea32a507ea32
        .octa     0x6fab95376fab95376fab95376fab9537
        .octa     0x6fab95376fab95376fab95376fab9537
        .octa     0x17406110174061101740611017406110
        .octa     0x17406110174061101740611017406110
        .octa     0x0d8cd6f10d8cd6f10d8cd6f10d8cd6f1
        .octa     0x0d8cd6f10d8cd6f10d8cd6f10d8cd6f1
        .octa     0xcdaa3b6dcdaa3b6dcdaa3b6dcdaa3b6d
        .octa     0xcdaa3b6dcdaa3b6dcdaa3b6dcdaa3b6d
        .octa     0xc0bbbe37c0bbbe37c0bbbe37c0bbbe37
        .octa     0xc0bbbe37c0bbbe37c0bbbe37c0bbbe37
        .octa     0x83613bda83613bda83613bda83613bda
        .octa     0x83613bda83613bda83613bda83613bda
        .octa     0xdb48a363db48a363db48a363db48a363
        .octa     0xdb48a363db48a363db48a363db48a363
        .octa     0x0b02e9310b02e9310b02e9310b02e931
        .octa     0x0b02e9310b02e9310b02e9310b02e931
        .octa     0x6fd15ca76fd15ca76fd15ca76fd15ca7
        .octa     0x6fd15ca76fd15ca76fd15ca76fd15ca7
        .octa     0x521afaca521afaca521afaca521afaca
        .octa     0x521afaca521afaca521afaca521afaca
        .octa     0x31338431313384313133843131338431
        .octa     0x31338431313384313133843131338431
        .octa     0x6ed41a956ed41a956ed41a956ed41a95
        .octa     0x6ed41a956ed41a956ed41a956ed41a95
        .octa     0x6d4378906d4378906d4378906d437890
        .octa     0x6d4378906d4378906d4378906d437890
        .octa     0xc39c91f2c39c91f2c39c91f2c39c91f2
        .octa     0xc39c91f2c39c91f2c39c91f2c39c91f2
        .octa     0x9eccabbd9eccabbd9eccabbd9eccabbd
        .octa     0x9eccabbd9eccabbd9eccabbd9eccabbd
        .octa     0xb5c9a0e6b5c9a0e6b5c9a0e6b5c9a0e6
        .octa     0xb5c9a0e6b5c9a0e6b5c9a0e6b5c9a0e6
        .octa     0x532fb63c532fb63c532fb63c532fb63c
        .octa     0x532fb63c532fb63c532fb63c532fb63c
        .octa     0xd2c741c6d2c741c6d2c741c6d2c741c6
        .octa     0xd2c741c6d2c741c6d2c741c6d2c741c6
        .octa     0x07237ea307237ea307237ea307237ea3
        .octa     0x07237ea307237ea307237ea307237ea3
        .octa     0xa4954b68a4954b68a4954b68a4954b68
        .octa     0xa4954b68a4954b68a4954b68a4954b68
        .octa     0x4c191d764c191d764c191d764c191d76
        .octa     0x4c191d764c191d764c191d764c191d76


.LDIGEST_8:
        .long   0x6a09e667, 0x6a09e667, 0x6a09e667, 0x6a09e667
        .long   0x6a09e667, 0x6a09e667, 0x6a09e667, 0x6a09e667
	.long 	0xbb67ae85, 0xbb67ae85, 0xbb67ae85, 0xbb67ae85
	.long 	0xbb67ae85, 0xbb67ae85, 0xbb67ae85, 0xbb67ae85
	.long   0x3c6ef372, 0x3c6ef372, 0x3c6ef372, 0x3c6ef372
	.long   0x3c6ef372, 0x3c6ef372, 0x3c6ef372, 0x3c6ef372
	.long 	0xa54ff53a, 0xa54ff53a, 0xa54ff53a, 0xa54ff53a
	.long 	0xa54ff53a, 0xa54ff53a, 0xa54ff53a, 0xa54ff53a
	.long	0x510e527f, 0x510e527f, 0x510e527f, 0x510e527f
	.long	0x510e527f, 0x510e527f, 0x510e527f, 0x510e527f
	.long 	0x9b05688c, 0x9b05688c, 0x9b05688c, 0x9b05688c
	.long 	0x9b05688c, 0x9b05688c, 0x9b05688c, 0x9b05688c
	.long	0x1f83d9ab, 0x1f83d9ab, 0x1f83d9ab, 0x1f83d9ab
	.long	0x1f83d9ab, 0x1f83d9ab, 0x1f83d9ab, 0x1f83d9ab
        .long   0x5be0cd19, 0x5be0cd19, 0x5be0cd19, 0x5be0cd19
        .long   0x5be0cd19, 0x5be0cd19, 0x5be0cd19, 0x5be0cd19


.LPSHUFFLE_BYTE_FLIP_MASK:
	.quad 0x0405060700010203, 0x0c0d0e0f08090a0b
	.quad 0x0405060700010203, 0x0c0d0e0f08090a0b


#ifdef __WIN64__
	.equiv OUTPUT_PTR,	rcx 	// 1st arg
	.equiv DATA_PTR,	rdx 	// 2nd arg
	.equiv NUM_BLKS, 	r8	// 3rd arg
	.equiv TBL, 		rsi
#else
	.equiv OUTPUT_PTR,	rdi	// 1st arg
	.equiv DATA_PTR,	rsi	// 2nd arg
	.equiv NUM_BLKS,	rdx	// 3rd arg
	.equiv TBL, 		rcx
#endif

.equiv ROUND,	rax
.equiv inp0, r9
.equiv inp1, r10
.equiv inp2, r11
.equiv inp3, r12
.equiv inp4, r13
.equiv inp5, r14

.eqv a, ymm0
.eqv b, ymm1
.eqv c, ymm2
.eqv d, ymm3
.eqv e, ymm4
.eqv f, ymm5
.eqv g, ymm6
.eqv h, ymm7

.equiv T1 , ymm8
.equiv a0, ymm12
.equiv a1, ymm13
.equiv a2, ymm14
.equiv TMP, ymm15
.equiv TMP0, ymm6
.equiv TMP1, ymm7
.equiv TT0, ymm8
.equiv TT1, ymm9
.equiv TT2, ymm10
.equiv TT3, ymm11
.equiv TT4, ymm12
.equiv TT5, ymm13
.equiv TT6, ymm14
.equiv TT7, ymm15

#define SHA256_DIGEST_WORD_SIZE  4
#define SZ8 32
#define ROUNDS 2048


#define _DIGEST	512
#define _YTMP	768
#define _RSAVE  896

#define YTMP0	rsp + _YTMP + 0*SZ8
#define YTMP1	rsp + _YTMP + 1*SZ8
#define YTMP2	rsp + _YTMP + 2*SZ8
#define YTMP3	rsp + _YTMP + 3*SZ8
#define R12 	rsp + _RSAVE + 0*8
#define R13 	rsp + _RSAVE + 1*8
#define R14 	rsp + _RSAVE + 2*8
#define R15 	rsp + _RSAVE + 3*8
#ifdef __WIN64__
	#define	_YMM_SAVE	928
	#define	FRAMESZ		1248
#else
	#define	FRAMESZ		928
#endif
#define VMOVPS	vmovups

.macro ROTATE_ARGS
.eqv TMP_, h
.eqv h, g
.eqv g, f
.eqv f, e
.eqv e, d
.eqv d, c
.eqv c, b
.eqv b, a
.eqv a, TMP_
.endm

.macro TRANSPOSE8_U32_LOAD8 r0, r1, r2, r3, r4, r5, r6, r7, addr0, addr1, addr2,\
                                        addr3, addr4, addr5, addr6, addr7,\
                                        ptr_offset
// Expected output data
//
// r0 = {e3 e2 e1 e0  a3 a2 a1 a0}
// r1 = {f3 f2 f1 f0  b3 b2 b1 b0}
// r2 = {g3 g2 g1 g0  c3 c2 c1 c0}
// r3 = {h3 h2 h1 h0  d3 d2 d1 d0}
// r4 = {e7 e6 e5 e4  a7 a6 a5 a4}
// r5 = {f7 f6 f5 f4  b7 b6 b5 b4}
// r6 = {g7 g6 g5 g4  c7 c6 c5 c4}
// r7 = {h7 h6 h5 h4  d7 d6 d5 d4}

	vmovups	\r0,[\addr0+\ptr_offset]
	vmovups	\r1,[\addr1+\ptr_offset]
	vmovups	\r2,[\addr2+\ptr_offset]
	vmovups	\r3,[\addr3+\ptr_offset]
	vmovups	\r4,[\addr0+\ptr_offset+16]
	vmovups	\r5,[\addr1+\ptr_offset+16]
	vmovups	\r6,[\addr2+\ptr_offset+16]
	vmovups	\r7,[\addr3+\ptr_offset+16]

	vinserti128 \r0, \r0, [\addr4+\ptr_offset], 0x01
	vinserti128 \r1, \r1, [\addr5+\ptr_offset], 0x01
	vinserti128 \r2, \r2, [\addr6+\ptr_offset], 0x01
	vinserti128 \r3, \r3, [\addr7+\ptr_offset], 0x01
	vinserti128 \r4, \r4, [\addr4+\ptr_offset+16], 0x01
	vinserti128 \r5, \r5, [\addr5+\ptr_offset+16], 0x01
	vinserti128 \r6, \r6, [\addr6+\ptr_offset+16], 0x01
	vinserti128 \r7, \r7, [\addr7+\ptr_offset+16], 0x01
.endm

.macro TRANSPOSE8_U32_PRELOADED r0, r1, r2, r3, r4, r5, r6, r7, t0, t1
# Input looks like: {r0 r1 r2 r3 r4 r5 r6 r7}
# r0 = {e3 e2 e1 e0   a3 a2 a1 a0}
# r1 = {f3 f2 f1 f0   b3 b2 b1 b0}
# r2 = {g3 g2 g1 g0   c3 c2 c1 c0}
# r3 = {h3 h2 h1 h0   d3 d2 d1 d0}
# r4 = {e7 e6 e5 e4   a7 a6 a5 a4}
# r5 = {f7 f6 f5 f4   b7 b6 b5 b4}
# r6 = {g7 g6 g5 g4   c7 c6 c5 c4}
# r7 = {h7 h6 h5 h4   d7 d6 d5 d4}
# Output looks like: {r0 r1 r2 r3 r4 r5 r6 r7}
# r0 = {h0 g0 f0 e0   d0 c0 b0 a0}
# r1 = {h1 g1 f1 e1   d1 c1 b1 a1}
# r2 = {h2 g2 f2 e2   d2 c2 b2 a2}
# r3 = {h3 g3 f3 e3   d3 c3 b3 a3}
# r4 = {h4 g4 f4 e4   d4 c4 b4 a4}
# r5 = {h5 g5 f5 e5   d5 c5 b5 a5}
# r6 = {h6 g6 f6 e6   d6 c6 b6 a6}
# r7 = {h7 g7 f7 e7   d7 c7 b7 a7}
# process top half (r0..r3)
	vshufps	\t0, \r0, \r1, 0x44	// t0 = {f1 f0 e1 e0   b1 b0 a1 a0}
	vshufps	\r0, \r0, \r1, 0xEE	// r0 = {f3 f2 e3 e2   b3 b2 a3 a2}
	vshufps \t1, \r2, \r3, 0x44	// t1 = {h1 h0 g1 g0   d1 d0 c1 c0}
	vshufps	\r2, \r2, \r3, 0xEE	// r2 = {h3 h2 g3 g2   d3 d2 c3 c2}

	vshufps	\r1, \t0, \t1, 0xDD	// r1 = {h1 g1 f1 e1   d1 c1 b1 a1}
	vshufps	\r3, \r0, \r2, 0xDD	// r3 = {h3 g3 f3 e3   d3 c3 b3 a3}
	vshufps	\r2, \r0, \r2, 0x88	// r2 = {h2 g2 f2 e2   d2 c2 b2 a2}
	vshufps	\r0, \t0, \t1, 0x88	// r0 = {h0 g0 f0 e0   d0 c0 b0 a0}

# process bottom half (r4..r7)
	vshufps	\t0, \r4, \r5, 0x44	// t0 = {f5 f4 e5 e4   b5 b4 a5 a4}
	vshufps	\r4, \r4, \r5, 0xEE	// r4 = {f7 f6 e7 e6   b7 b6 a7 a6}
	vshufps \t1, \r6, \r7, 0x44	// t1 = {h5 h4 g5 g4   d5 d4 c5 c4}
	vshufps	\r6, \r6, \r7, 0xEE	// r6 = {h7 h6 g7 g6   d7 d6 c7 c6}

	vshufps	\r5, \t0, \t1, 0xDD	// r5 = {h5 g5 f5 e5   d5 c5 b5 a5}
	vshufps	\r7, \r4, \r6, 0xDD	// r7 = {h7 g7 f7 e7   d7 c7 b7 a7}
	vshufps	\r6, \r4, \r6, 0x88	// r6 = {h6 g6 f6 e6   d6 c6 b6 a6}
	vshufps	\r4, \t0, \t1, 0x88	// r4 = {h4 g4 f4 e4   d4 c4 b4 a4}
.endm

.macro TRANSPOSE8_U32 r0, r1, r2, r3, r4, r5, r6, r7, t0, t1
       # process top half (r0..r3) {a...d}
       vshufps \t0, \r0, \r1, 0x44  // t0 = {b5 b4 a5 a4   b1 b0 a1 a0}
       vshufps \r0, \r0, \r1, 0xEE  // r0 = {b7 b6 a7 a6   b3 b2 a3 a2}
       vshufps \t1, \r2, \r3, 0x44  // t1 = {d5 d4 c5 c4   d1 d0 c1 c0}
       vshufps \r2, \r2, \r3, 0xEE  // r2 = {d7 d6 c7 c6   d3 d2 c3 c2}
       vshufps \r3, \t0, \t1, 0xDD  // r3 = {d5 c5 b5 a5   d1 c1 b1 a1}
       vshufps \r1, \r0, \r2, 0x88  // r1 = {d6 c6 b6 a6   d2 c2 b2 a2}
       vshufps \r0, \r0, \r2, 0xDD  // r0 = {d7 c7 b7 a7   d3 c3 b3 a3}
       vshufps \t0, \t0, \t1, 0x88  // t0 = {d4 c4 b4 a4   d0 c0 b0 a0}

       # use r2 in place of t0
       # process bottom half (r4..r7) {e...h}
       vshufps \r2, \r4, \r5, 0x44  // r2 = {f5 f4 e5 e4   f1 f0 e1 e0}
       vshufps \r4, \r4, \r5, 0xEE  // r4 = {f7 f6 e7 e6   f3 f2 e3 e2}
       vshufps \t1, \r6, \r7, 0x44  // t1 = {h5 h4 g5 g4   h1 h0 g1 g0}
       vshufps \r6, \r6, \r7, 0xEE  // r6 = {h7 h6 g7 g6   h3 h2 g3 g2}
       vshufps \r7, \r2, \t1, 0xDD  // r7 = {h5 g5 f5 e5   h1 g1 f1 e1}
       vshufps \r5, \r4, \r6, 0x88  // r5 = {h6 g6 f6 e6   h2 g2 f2 e2}
       vshufps \r4, \r4, \r6, 0xDD  // r4 = {h7 g7 f7 e7   h3 g3 f3 e3}
       vshufps \t1, \r2, \t1, 0x88  // t1 = {h4 g4 f4 e4   h0 g0 f0 e0}


       vperm2f128      \r6, \r5, \r1, 0x13  // h6...a6
       vperm2f128      \r2, \r5, \r1, 0x02  // h2...a2
       vperm2f128      \r5, \r7, \r3, 0x13  // h5...a5
       vperm2f128      \r1, \r7, \r3, 0x02  // h1...a1
       vperm2f128      \r7, \r4, \r0, 0x13  // h7...a7
       vperm2f128      \r3, \r4, \r0, 0x02  // h3...a3
       vperm2f128      \r4, \t1, \t0, 0x13  // h4...a4
       vperm2f128      \r0, \t1, \t0, 0x02  // h0...a0
.endm

.macro PRORD3 reg, imm, tmp
	vpslld	\tmp, \reg, (32-(\imm))
	vpsrld	\reg, \reg, \imm
	vpor	\reg, \reg, \tmp
.endm

.macro PRORD_nd4 reg, imm, tmp, src
	vpslld	\tmp, \src, (32-(\imm))
	vpsrld	\reg, \src, \imm
	vpor	\reg, \reg, \tmp
.endm

.macro PRORD src, imm
	PRORD3	\src, \imm, TMP
.endm

.macro PRORD_nd dst, src, amt
	PRORD_nd4	\dst, \amt, TMP, \src
.endm

.macro ROUND_00_15 T1, i
	PRORD_nd	a0, e, (11-6)	// sig1: a0 = (e >> 5)

	vpxor	a2, f, g	// ch: a2 = f^g
	vpand	a2, a2, e		// ch: a2 = (f^g)&e
	vpxor	a2, a2, g		// a2 = ch

	PRORD_nd	a1, e, 25		// sig1: a1 = (e >> 25)
.if .Lpadding - 1
	vmovdqa	[SZ8*(\i&0xf) + rsp], \T1     // save current temp message
	vpaddd	\T1, \T1, [TBL + ROUND]	// T1 = W + K
.else
	vmovdqa 	\T1, [TBL + ROUND]	// T1 = W + K
.endif
	vpxor	a0, a0, e	// sig1: a0 = e ^ (e >> 5)
	PRORD	a0, 6		// sig1: a0 = (e >> 6) ^ (e >> 11)
	vpaddd	h, h, a2	// h = h + ch
	PRORD_nd	a2, a, (13-2)	// sig0: a2 = (a >> 11)
	vpaddd	h, h, \T1	// h = h + ch + W + K
	vpxor	a0, a0, a1	// a0 = sigma1
	PRORD_nd	a1, a, 22	// sig0: a1 = (a >> 22)
	vpxor	\T1, a, c	// maj: T1 = a^c
	add	ROUND, SZ8	// ROUND++
	vpand	\T1, \T1, b	// maj: T1 = (a^c)&b
	vpaddd	h, h, a0

	vpaddd	d, d, h

	vpxor	a2, a2, a	// sig0: a2 = a ^ (a >> 11)
	PRORD	a2, 2		// sig0: a2 = (a >> 2) ^ (a >> 13)
	vpxor	a2, a2, a1	// a2 = sig0
	vpand	a1, a, c	// maj: a1 = a&c
	vpor	a1, a1, \T1	// a1 = maj
	vpaddd	h, h, a1	// h = h + ch + W + K + maj
	vpaddd	h, h, a2	// h = h + ch + W + K + maj + sigma0
	ROTATE_ARGS
.endm

.macro ROUND_16_XX T1, i
	vmovdqa	\T1, [SZ8*((\i-15)&0xf) + rsp]
	vmovdqa	a1, [SZ8*((\i-2)&0xf) + rsp]
	vmovdqa	a0, \T1
	PRORD	\T1, 18-7
	vmovdqa	a2, a1
	PRORD	a1, 19-17
	vpxor	\T1, \T1, a0
	PRORD	\T1, 7
	vpxor	a1, a1, a2
	PRORD	a1, 17
	vpsrld	a0, a0, 3
	vpxor	\T1, \T1, a0
	vpsrld	a2, a2, 10
	vpxor	a1, a1, a2
	vpaddd	\T1, \T1, [SZ8*((\i-16)&0xf) + rsp]   // + W[i-16]
	vpaddd	a1, a1, [SZ8*((\i-7)&0xf) + rsp]        // + W[i-7]
	vpaddd	\T1, \T1, a1

	ROUND_00_15 \T1, \i
.endm

.text
.global sha256_8_avx2
#ifndef __WIN64__
.type   sha256_8_avx2,%function
#endif
.align 16
sha256_8_avx2:
        endbr64
        cmp     NUM_BLKS, 0
        jne     .Lstart_routine
        ret
.Lstart_routine:
	push 	rbp
	mov     rbp,rsp
	and 	rsp, ~31
	sub	rsp, FRAMESZ
	mov	[R12], r12
	mov	[R13], r13
	mov	[R14], r14
	mov	[R15], r15
#ifdef __WIN64__
	vmovdqa	[rsp + _YMM_SAVE + 0*32],ymm6
	vmovdqa	[rsp + _YMM_SAVE + 1*32],ymm7
	vmovdqa	[rsp + _YMM_SAVE + 2*32],ymm8
	vmovdqa	[rsp + _YMM_SAVE + 3*32],ymm9
	vmovdqa	[rsp + _YMM_SAVE + 4*32],ymm10
	vmovdqa	[rsp + _YMM_SAVE + 5*32],ymm11
	vmovdqa	[rsp + _YMM_SAVE + 6*32],ymm12
	vmovdqa	[rsp + _YMM_SAVE + 7*32],ymm13
	vmovdqa	[rsp + _YMM_SAVE + 8*32],ymm14
	vmovdqa	[rsp + _YMM_SAVE + 9*32],ymm15
#endif


.Lsha256_8_avx2_loop:
.set .Lpadding, 0
	cmp 	NUM_BLKS, 8
	jb 	.Lsha256_8_avx2_epilog
	xor	ROUND, ROUND

	lea TBL,[rip + .LDIGEST_8]
	vmovdqa	a,[TBL + 0*32]
	vmovdqa	b,[TBL + 1*32]
	vmovdqa	c,[TBL + 2*32]
	vmovdqa	d,[TBL + 3*32]
	vmovdqa	e,[TBL + 4*32]
	vmovdqa	f,[TBL + 5*32]
	vmovdqa	g,[TBL + 6*32]
	vmovdqa	h,[TBL + 7*32]

	lea	TBL,[rip + .LK256_8]

.set .Li, 0
.rept 2
	TRANSPOSE8_U32_LOAD8 TT0, TT1, TT2, TT3, TT4, TT5, TT6, TT7, \
			     DATA_PTR + 0*64, \
			     DATA_PTR + 1*64, \
			     DATA_PTR + 2*64, \
			     DATA_PTR + 3*64, \
			     DATA_PTR + 4*64, \
			     DATA_PTR + 5*64, \
			     DATA_PTR + 6*64, \
			     DATA_PTR + 7*64, \
			     32*.Li

	vmovdqa	[YTMP0], g
	vmovdqa	[YTMP1], h
	TRANSPOSE8_U32_PRELOADED TT0, TT1, TT2, TT3, TT4, TT5, TT6, TT7, TMP0, TMP1
	vmovdqa	TMP1, [rip + .LPSHUFFLE_BYTE_FLIP_MASK]
	vmovdqa	g, [YTMP0]
	vpshufb	TT0, TT0, TMP1
	vpshufb	TT1, TT1, TMP1
	vpshufb	TT2, TT2, TMP1
	vpshufb	TT3, TT3, TMP1
	vpshufb	TT4, TT4, TMP1
	vpshufb	TT5, TT5, TMP1
	vpshufb	TT6, TT6, TMP1
	vpshufb	TT7, TT7, TMP1
	vmovdqa	h, [YTMP1]
	vmovdqa	[YTMP0], TT4
	vmovdqa	[YTMP1], TT5
	vmovdqa	[YTMP2], TT6
	vmovdqa	[YTMP3], TT7
	ROUND_00_15	TT0, 8*.Li
	vmovdqa	TT0, [YTMP0]
	ROUND_00_15	TT1, 8*.Li+1
	vmovdqa	TT1, [YTMP1]
	ROUND_00_15	TT2, 8*.Li+2
	vmovdqa	TT2, [YTMP2]
	ROUND_00_15	TT3, 8*.Li+3
	vmovdqa	TT3, [YTMP3]
	ROUND_00_15	TT0, 8*.Li+4
	ROUND_00_15	TT1, 8*.Li+5
	ROUND_00_15	TT2, 8*.Li+6
	ROUND_00_15	TT3, 8*.Li+7
.set .Li, .Li+1
.endr

.set .Li, 8*.Li
	jmp	.Lrounds_16_xx
.align 16
.Lrounds_16_xx:
.rept 16
	ROUND_16_XX	T1, .Li
.set .Li, (.Li+1)
.endr

	cmp	ROUND,ROUNDS
	jb	.Lrounds_16_xx

	# add old digest
	lea TBL,[rip + .LDIGEST_8]
	vpaddd	a, a, [TBL + 0*SZ8]
	vpaddd	b, b, [TBL + 1*SZ8]
	vpaddd	c, c, [TBL + 2*SZ8]
	vpaddd	d, d, [TBL + 3*SZ8]
	vpaddd	e, e, [TBL + 4*SZ8]
	vpaddd	f, f, [TBL + 5*SZ8]
	vpaddd	g, g, [TBL + 6*SZ8]
	vpaddd	h, h, [TBL + 7*SZ8]

	# rounds with padding

	# save old digest
	vmovdqa	[rsp + _DIGEST + 0*SZ8], a
	vmovdqa	[rsp + _DIGEST + 1*SZ8], b
	vmovdqa	[rsp + _DIGEST + 2*SZ8], c
	vmovdqa	[rsp + _DIGEST + 3*SZ8], d
	vmovdqa	[rsp + _DIGEST + 4*SZ8], e
	vmovdqa	[rsp + _DIGEST + 5*SZ8], f
	vmovdqa	[rsp + _DIGEST + 6*SZ8], g
	vmovdqa	[rsp + _DIGEST + 7*SZ8], h


	lea TBL,[rip + .LPADDING_8]
	xor ROUND,ROUND
	jmp 	.Lrounds_padding

.set .Lpadding, 1
.align 16
.Lrounds_padding:
.rept 64
	ROUND_00_15 	T1, 0
.endr
	# add old digest
	vpaddd	a, a, [rsp + _DIGEST + 0*SZ8]
	vpaddd	b, b, [rsp + _DIGEST + 1*SZ8]
	vpaddd	c, c, [rsp + _DIGEST + 2*SZ8]
	vpaddd	d, d, [rsp + _DIGEST + 3*SZ8]
	vpaddd	e, e, [rsp + _DIGEST + 4*SZ8]
	vpaddd	f, f, [rsp + _DIGEST + 5*SZ8]
	vpaddd	g, g, [rsp + _DIGEST + 6*SZ8]
	vpaddd	h, h, [rsp + _DIGEST + 7*SZ8]


	# transpose the digest and convert to little endian to get the registers correctly

	TRANSPOSE8_U32 a, b, c, d, e, f, g, h, TT0, TT1
	vmovdqa	TT0, [rip + .LPSHUFFLE_BYTE_FLIP_MASK]
	vpshufb	a, a, TT0
	vpshufb	b, b, TT0
	vpshufb	c, c, TT0
	vpshufb	d, d, TT0
	vpshufb	e, e, TT0
	vpshufb	f, f, TT0
	vpshufb	g, g, TT0
	vpshufb	h, h, TT0

	# write to output

	vmovdqu	[OUTPUT_PTR + 0*32],a
	vmovdqu	[OUTPUT_PTR + 1*32],b
	vmovdqu	[OUTPUT_PTR + 2*32],c
	vmovdqu	[OUTPUT_PTR + 3*32],d
	vmovdqu	[OUTPUT_PTR + 4*32],e
	vmovdqu	[OUTPUT_PTR + 5*32],f
	vmovdqu	[OUTPUT_PTR + 6*32],g
	vmovdqu	[OUTPUT_PTR + 7*32],h

	# update pointers and loop

        add 	DATA_PTR, 64*8
	add 	OUTPUT_PTR, 32*8
	sub 	NUM_BLKS, 8

	jmp     .Lsha256_8_avx2_loop

.Lsha256_8_avx2_epilog:
#ifdef __WIN64__
	vmovdqa	ymm6,[rsp + _YMM_SAVE + 0*32]
	vmovdqa	ymm7,[rsp + _YMM_SAVE + 1*32]
	vmovdqa	ymm8,[rsp + _YMM_SAVE + 2*32]
	vmovdqa	ymm9,[rsp + _YMM_SAVE + 3*32]
	vmovdqa	ymm10,[rsp + _YMM_SAVE + 4*32]
	vmovdqa	ymm11,[rsp + _YMM_SAVE + 5*32]
	vmovdqa	ymm12,[rsp + _YMM_SAVE + 6*32]
	vmovdqa	ymm13,[rsp + _YMM_SAVE + 7*32]
	vmovdqa	ymm14,[rsp + _YMM_SAVE + 8*32]
	vmovdqa	ymm15,[rsp + _YMM_SAVE + 9*32]
#endif
	mov	r12,[R12]
	mov	r13,[R13]
	mov	r14,[R14]
	mov	r15,[R15]

	mov     rsp,rbp
	pop     rbp
        jmp     sha256_4_avx

#ifdef __linux__
.size sha256_8_avx2,.-sha256_8_avx2
.section .note.GNU-stack,"",@progbits
#endif
#endif
