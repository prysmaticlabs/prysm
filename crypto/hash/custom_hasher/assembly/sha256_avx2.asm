;;  sha256_avx2.asm
; *
; *  This file is part of Mammon.
; *  mammon is a greedy and selfish ETH consensus client.
; *
; *  Copyright (c) 2021 - Reimundo Heluani (potuz) potuz@potuz.net
; *
; *  This program is free software: you can redistribute it and/or modify
; *  it under the terms of the GNU General Public License as published by
; *  the Free Software Foundation, either version 3 of the License, or
; *  (at your option) any later version.
; *
; *  This program is distributed in the hope that it will be useful,
; *  but WITHOUT ANY WARRANTY; without even the implied warranty of
; *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
; *  GNU General Public License for more details.
; *
;  You should have received a copy of the GNU General Public License
;  along with this program.  If not, see <http://www.gnu.org/licenses/>.
;
;  This implementation is a 64 bytes optimized implementation based on Intel's code
;  whose copyright follows
;
;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
;;
;; Copyright (c) 2012-2021, Intel Corporation
;;
;; Redistribution and use in source and binary forms, with or without
;; modification, are permitted provided that the following conditions are met:
;;
;;     * Redistributions of source code must retain the above copyright notice,
;;       this list of conditions and the following disclaimer.
;;     * Redistributions in binary form must reproduce the above copyright
;;       notice, this list of conditions and the following disclaimer in the
;;       documentation and/or other materials provided with the distribution.
;;     * Neither the name of Intel Corporation nor the names of its contributors
;;       may be used to endorse or promote products derived from this software
;;       without specific prior written permission.
;;
;; THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
;; AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
;; IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
;; DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE
;; FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
;; DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
;; SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
;; CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
;; OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
;; OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
;;

;; code to compute oct SHA256 using SSE-256
;; outer calling routine takes care of save and restore of XMM registers
;; Logic designed/laid out by JDG

;; Function clobbers: rax, rcx, rdx, rsi, rdi, r9-r15; ymm0-15
;; Stack must be aligned to 32 bytes before call
;; Windows clobbers:  rax rdx rsi rdi     r8 r9 r10 r11 r12 r13 r14
;; Windows preserves:         rcx             rbp                           r15
;;
;; Linux clobbers:    rax rcx rdx rsi         r8 r9 r10 r11 r12 r13 r14
;; Linux preserves:                       rdi rbp                           r15
;;
;; clobbers ymm0-15

%include "transpose_avx2.asm"

extern sha256_4_avx

section .data
default rel
align 64

K256_8:
	dq	0x428a2f98428a2f98, 0x428a2f98428a2f98
	dq	0x428a2f98428a2f98, 0x428a2f98428a2f98
	dq	0x7137449171374491, 0x7137449171374491
	dq	0x7137449171374491, 0x7137449171374491
	dq	0xb5c0fbcfb5c0fbcf, 0xb5c0fbcfb5c0fbcf
	dq	0xb5c0fbcfb5c0fbcf, 0xb5c0fbcfb5c0fbcf
	dq	0xe9b5dba5e9b5dba5, 0xe9b5dba5e9b5dba5
	dq	0xe9b5dba5e9b5dba5, 0xe9b5dba5e9b5dba5
	dq	0x3956c25b3956c25b, 0x3956c25b3956c25b
	dq	0x3956c25b3956c25b, 0x3956c25b3956c25b
	dq	0x59f111f159f111f1, 0x59f111f159f111f1
	dq	0x59f111f159f111f1, 0x59f111f159f111f1
	dq	0x923f82a4923f82a4, 0x923f82a4923f82a4
	dq	0x923f82a4923f82a4, 0x923f82a4923f82a4
	dq	0xab1c5ed5ab1c5ed5, 0xab1c5ed5ab1c5ed5
	dq	0xab1c5ed5ab1c5ed5, 0xab1c5ed5ab1c5ed5
	dq	0xd807aa98d807aa98, 0xd807aa98d807aa98
	dq	0xd807aa98d807aa98, 0xd807aa98d807aa98
	dq	0x12835b0112835b01, 0x12835b0112835b01
	dq	0x12835b0112835b01, 0x12835b0112835b01
	dq	0x243185be243185be, 0x243185be243185be
	dq	0x243185be243185be, 0x243185be243185be
	dq	0x550c7dc3550c7dc3, 0x550c7dc3550c7dc3
	dq	0x550c7dc3550c7dc3, 0x550c7dc3550c7dc3
	dq	0x72be5d7472be5d74, 0x72be5d7472be5d74
	dq	0x72be5d7472be5d74, 0x72be5d7472be5d74
	dq	0x80deb1fe80deb1fe, 0x80deb1fe80deb1fe
	dq	0x80deb1fe80deb1fe, 0x80deb1fe80deb1fe
	dq	0x9bdc06a79bdc06a7, 0x9bdc06a79bdc06a7
	dq	0x9bdc06a79bdc06a7, 0x9bdc06a79bdc06a7
	dq	0xc19bf174c19bf174, 0xc19bf174c19bf174
	dq	0xc19bf174c19bf174, 0xc19bf174c19bf174
	dq	0xe49b69c1e49b69c1, 0xe49b69c1e49b69c1
	dq	0xe49b69c1e49b69c1, 0xe49b69c1e49b69c1
	dq	0xefbe4786efbe4786, 0xefbe4786efbe4786
	dq	0xefbe4786efbe4786, 0xefbe4786efbe4786
	dq	0x0fc19dc60fc19dc6, 0x0fc19dc60fc19dc6
	dq	0x0fc19dc60fc19dc6, 0x0fc19dc60fc19dc6
	dq	0x240ca1cc240ca1cc, 0x240ca1cc240ca1cc
	dq	0x240ca1cc240ca1cc, 0x240ca1cc240ca1cc
	dq	0x2de92c6f2de92c6f, 0x2de92c6f2de92c6f
	dq	0x2de92c6f2de92c6f, 0x2de92c6f2de92c6f
	dq	0x4a7484aa4a7484aa, 0x4a7484aa4a7484aa
	dq	0x4a7484aa4a7484aa, 0x4a7484aa4a7484aa
	dq	0x5cb0a9dc5cb0a9dc, 0x5cb0a9dc5cb0a9dc
	dq	0x5cb0a9dc5cb0a9dc, 0x5cb0a9dc5cb0a9dc
	dq	0x76f988da76f988da, 0x76f988da76f988da
	dq	0x76f988da76f988da, 0x76f988da76f988da
	dq	0x983e5152983e5152, 0x983e5152983e5152
	dq	0x983e5152983e5152, 0x983e5152983e5152
	dq	0xa831c66da831c66d, 0xa831c66da831c66d
	dq	0xa831c66da831c66d, 0xa831c66da831c66d
	dq	0xb00327c8b00327c8, 0xb00327c8b00327c8
	dq	0xb00327c8b00327c8, 0xb00327c8b00327c8
	dq	0xbf597fc7bf597fc7, 0xbf597fc7bf597fc7
	dq	0xbf597fc7bf597fc7, 0xbf597fc7bf597fc7
	dq	0xc6e00bf3c6e00bf3, 0xc6e00bf3c6e00bf3
	dq	0xc6e00bf3c6e00bf3, 0xc6e00bf3c6e00bf3
	dq	0xd5a79147d5a79147, 0xd5a79147d5a79147
	dq	0xd5a79147d5a79147, 0xd5a79147d5a79147
	dq	0x06ca635106ca6351, 0x06ca635106ca6351
	dq	0x06ca635106ca6351, 0x06ca635106ca6351
	dq	0x1429296714292967, 0x1429296714292967
	dq	0x1429296714292967, 0x1429296714292967
	dq	0x27b70a8527b70a85, 0x27b70a8527b70a85
	dq	0x27b70a8527b70a85, 0x27b70a8527b70a85
	dq	0x2e1b21382e1b2138, 0x2e1b21382e1b2138
	dq	0x2e1b21382e1b2138, 0x2e1b21382e1b2138
	dq	0x4d2c6dfc4d2c6dfc, 0x4d2c6dfc4d2c6dfc
	dq	0x4d2c6dfc4d2c6dfc, 0x4d2c6dfc4d2c6dfc
	dq	0x53380d1353380d13, 0x53380d1353380d13
	dq	0x53380d1353380d13, 0x53380d1353380d13
	dq	0x650a7354650a7354, 0x650a7354650a7354
	dq	0x650a7354650a7354, 0x650a7354650a7354
	dq	0x766a0abb766a0abb, 0x766a0abb766a0abb
	dq	0x766a0abb766a0abb, 0x766a0abb766a0abb
	dq	0x81c2c92e81c2c92e, 0x81c2c92e81c2c92e
	dq	0x81c2c92e81c2c92e, 0x81c2c92e81c2c92e
	dq	0x92722c8592722c85, 0x92722c8592722c85
	dq	0x92722c8592722c85, 0x92722c8592722c85
	dq	0xa2bfe8a1a2bfe8a1, 0xa2bfe8a1a2bfe8a1
	dq	0xa2bfe8a1a2bfe8a1, 0xa2bfe8a1a2bfe8a1
	dq	0xa81a664ba81a664b, 0xa81a664ba81a664b
	dq	0xa81a664ba81a664b, 0xa81a664ba81a664b
	dq	0xc24b8b70c24b8b70, 0xc24b8b70c24b8b70
	dq	0xc24b8b70c24b8b70, 0xc24b8b70c24b8b70
	dq	0xc76c51a3c76c51a3, 0xc76c51a3c76c51a3
	dq	0xc76c51a3c76c51a3, 0xc76c51a3c76c51a3
	dq	0xd192e819d192e819, 0xd192e819d192e819
	dq	0xd192e819d192e819, 0xd192e819d192e819
	dq	0xd6990624d6990624, 0xd6990624d6990624
	dq	0xd6990624d6990624, 0xd6990624d6990624
	dq	0xf40e3585f40e3585, 0xf40e3585f40e3585
	dq	0xf40e3585f40e3585, 0xf40e3585f40e3585
	dq	0x106aa070106aa070, 0x106aa070106aa070
	dq	0x106aa070106aa070, 0x106aa070106aa070
	dq	0x19a4c11619a4c116, 0x19a4c11619a4c116
	dq	0x19a4c11619a4c116, 0x19a4c11619a4c116
	dq	0x1e376c081e376c08, 0x1e376c081e376c08
	dq	0x1e376c081e376c08, 0x1e376c081e376c08
	dq	0x2748774c2748774c, 0x2748774c2748774c
	dq	0x2748774c2748774c, 0x2748774c2748774c
	dq	0x34b0bcb534b0bcb5, 0x34b0bcb534b0bcb5
	dq	0x34b0bcb534b0bcb5, 0x34b0bcb534b0bcb5
	dq	0x391c0cb3391c0cb3, 0x391c0cb3391c0cb3
	dq	0x391c0cb3391c0cb3, 0x391c0cb3391c0cb3
	dq	0x4ed8aa4a4ed8aa4a, 0x4ed8aa4a4ed8aa4a
	dq	0x4ed8aa4a4ed8aa4a, 0x4ed8aa4a4ed8aa4a
	dq	0x5b9cca4f5b9cca4f, 0x5b9cca4f5b9cca4f
	dq	0x5b9cca4f5b9cca4f, 0x5b9cca4f5b9cca4f
	dq	0x682e6ff3682e6ff3, 0x682e6ff3682e6ff3
	dq	0x682e6ff3682e6ff3, 0x682e6ff3682e6ff3
	dq	0x748f82ee748f82ee, 0x748f82ee748f82ee
	dq	0x748f82ee748f82ee, 0x748f82ee748f82ee
	dq	0x78a5636f78a5636f, 0x78a5636f78a5636f
	dq	0x78a5636f78a5636f, 0x78a5636f78a5636f
	dq	0x84c8781484c87814, 0x84c8781484c87814
	dq	0x84c8781484c87814, 0x84c8781484c87814
	dq	0x8cc702088cc70208, 0x8cc702088cc70208
	dq	0x8cc702088cc70208, 0x8cc702088cc70208
	dq	0x90befffa90befffa, 0x90befffa90befffa
	dq	0x90befffa90befffa, 0x90befffa90befffa
	dq	0xa4506ceba4506ceb, 0xa4506ceba4506ceb
	dq	0xa4506ceba4506ceb, 0xa4506ceba4506ceb
	dq	0xbef9a3f7bef9a3f7, 0xbef9a3f7bef9a3f7
	dq	0xbef9a3f7bef9a3f7, 0xbef9a3f7bef9a3f7
	dq	0xc67178f2c67178f2, 0xc67178f2c67178f2
	dq	0xc67178f2c67178f2, 0xc67178f2c67178f2

PADDING_8:

	ddq     0xc28a2f98c28a2f98c28a2f98c28a2f98
	ddq     0xc28a2f98c28a2f98c28a2f98c28a2f98
	ddq 	0x71374491713744917137449171374491
	ddq 	0x71374491713744917137449171374491
	ddq 	0xb5c0fbcfb5c0fbcfb5c0fbcfb5c0fbcf
	ddq 	0xb5c0fbcfb5c0fbcfb5c0fbcfb5c0fbcf
	ddq 	0xe9b5dba5e9b5dba5e9b5dba5e9b5dba5
	ddq 	0xe9b5dba5e9b5dba5e9b5dba5e9b5dba5
        ddq     0x3956c25b3956c25b3956c25b3956c25b
        ddq     0x3956c25b3956c25b3956c25b3956c25b
        ddq     0x59f111f159f111f159f111f159f111f1
        ddq     0x59f111f159f111f159f111f159f111f1
        ddq     0x923f82a4923f82a4923f82a4923f82a4
        ddq     0x923f82a4923f82a4923f82a4923f82a4
        ddq     0xab1c5ed5ab1c5ed5ab1c5ed5ab1c5ed5
        ddq     0xab1c5ed5ab1c5ed5ab1c5ed5ab1c5ed5
        ddq     0xd807aa98d807aa98d807aa98d807aa98
        ddq     0xd807aa98d807aa98d807aa98d807aa98
        ddq     0x12835b0112835b0112835b0112835b01
        ddq     0x12835b0112835b0112835b0112835b01
        ddq     0x243185be243185be243185be243185be
        ddq     0x243185be243185be243185be243185be
        ddq     0x550c7dc3550c7dc3550c7dc3550c7dc3
        ddq     0x550c7dc3550c7dc3550c7dc3550c7dc3
        ddq     0x72be5d7472be5d7472be5d7472be5d74
        ddq     0x72be5d7472be5d7472be5d7472be5d74
        ddq     0x80deb1fe80deb1fe80deb1fe80deb1fe
        ddq     0x80deb1fe80deb1fe80deb1fe80deb1fe
        ddq     0x9bdc06a79bdc06a79bdc06a79bdc06a7
        ddq     0x9bdc06a79bdc06a79bdc06a79bdc06a7
        ddq     0xc19bf374c19bf374c19bf374c19bf374
        ddq     0xc19bf374c19bf374c19bf374c19bf374
        ddq     0x649b69c1649b69c1649b69c1649b69c1
        ddq     0x649b69c1649b69c1649b69c1649b69c1
        ddq     0xf0fe4786f0fe4786f0fe4786f0fe4786
        ddq     0xf0fe4786f0fe4786f0fe4786f0fe4786
        ddq     0x0fe1edc60fe1edc60fe1edc60fe1edc6
        ddq     0x0fe1edc60fe1edc60fe1edc60fe1edc6
        ddq     0x240cf254240cf254240cf254240cf254
        ddq     0x240cf254240cf254240cf254240cf254
        ddq     0x4fe9346f4fe9346f4fe9346f4fe9346f
        ddq     0x4fe9346f4fe9346f4fe9346f4fe9346f
        ddq     0x6cc984be6cc984be6cc984be6cc984be
        ddq     0x6cc984be6cc984be6cc984be6cc984be
        ddq     0x61b9411e61b9411e61b9411e61b9411e
        ddq     0x61b9411e61b9411e61b9411e61b9411e
        ddq     0x16f988fa16f988fa16f988fa16f988fa
        ddq     0x16f988fa16f988fa16f988fa16f988fa
        ddq     0xf2c65152f2c65152f2c65152f2c65152
        ddq     0xf2c65152f2c65152f2c65152f2c65152
        ddq     0xa88e5a6da88e5a6da88e5a6da88e5a6d
        ddq     0xa88e5a6da88e5a6da88e5a6da88e5a6d
        ddq     0xb019fc65b019fc65b019fc65b019fc65
        ddq     0xb019fc65b019fc65b019fc65b019fc65
        ddq     0xb9d99ec7b9d99ec7b9d99ec7b9d99ec7
        ddq     0xb9d99ec7b9d99ec7b9d99ec7b9d99ec7
        ddq     0x9a1231c39a1231c39a1231c39a1231c3
        ddq     0x9a1231c39a1231c39a1231c39a1231c3
        ddq     0xe70eeaa0e70eeaa0e70eeaa0e70eeaa0
        ddq     0xe70eeaa0e70eeaa0e70eeaa0e70eeaa0
        ddq     0xfdb1232bfdb1232bfdb1232bfdb1232b
        ddq     0xfdb1232bfdb1232bfdb1232bfdb1232b
        ddq     0xc7353eb0c7353eb0c7353eb0c7353eb0
        ddq     0xc7353eb0c7353eb0c7353eb0c7353eb0
        ddq     0x3069bad53069bad53069bad53069bad5
        ddq     0x3069bad53069bad53069bad53069bad5
        ddq     0xcb976d5fcb976d5fcb976d5fcb976d5f
        ddq     0xcb976d5fcb976d5fcb976d5fcb976d5f
        ddq     0x5a0f118f5a0f118f5a0f118f5a0f118f
        ddq     0x5a0f118f5a0f118f5a0f118f5a0f118f
        ddq     0xdc1eeefddc1eeefddc1eeefddc1eeefd
        ddq     0xdc1eeefddc1eeefddc1eeefddc1eeefd
        ddq     0x0a35b6890a35b6890a35b6890a35b689
        ddq     0x0a35b6890a35b6890a35b6890a35b689
        ddq     0xde0b7a04de0b7a04de0b7a04de0b7a04
        ddq     0xde0b7a04de0b7a04de0b7a04de0b7a04
        ddq     0x58f4ca9d58f4ca9d58f4ca9d58f4ca9d
        ddq     0x58f4ca9d58f4ca9d58f4ca9d58f4ca9d
        ddq     0xe15d5b16e15d5b16e15d5b16e15d5b16
        ddq     0xe15d5b16e15d5b16e15d5b16e15d5b16
        ddq     0x007f3e86007f3e86007f3e86007f3e86
        ddq     0x007f3e86007f3e86007f3e86007f3e86
        ddq     0x37088980370889803708898037088980
        ddq     0x37088980370889803708898037088980
        ddq     0xa507ea32a507ea32a507ea32a507ea32
        ddq     0xa507ea32a507ea32a507ea32a507ea32
        ddq     0x6fab95376fab95376fab95376fab9537
        ddq     0x6fab95376fab95376fab95376fab9537
        ddq     0x17406110174061101740611017406110
        ddq     0x17406110174061101740611017406110
        ddq     0x0d8cd6f10d8cd6f10d8cd6f10d8cd6f1
        ddq     0x0d8cd6f10d8cd6f10d8cd6f10d8cd6f1
        ddq     0xcdaa3b6dcdaa3b6dcdaa3b6dcdaa3b6d
        ddq     0xcdaa3b6dcdaa3b6dcdaa3b6dcdaa3b6d
        ddq     0xc0bbbe37c0bbbe37c0bbbe37c0bbbe37
        ddq     0xc0bbbe37c0bbbe37c0bbbe37c0bbbe37
        ddq     0x83613bda83613bda83613bda83613bda
        ddq     0x83613bda83613bda83613bda83613bda
        ddq     0xdb48a363db48a363db48a363db48a363
        ddq     0xdb48a363db48a363db48a363db48a363
        ddq     0x0b02e9310b02e9310b02e9310b02e931
        ddq     0x0b02e9310b02e9310b02e9310b02e931
        ddq     0x6fd15ca76fd15ca76fd15ca76fd15ca7
        ddq     0x6fd15ca76fd15ca76fd15ca76fd15ca7
        ddq     0x521afaca521afaca521afaca521afaca
        ddq     0x521afaca521afaca521afaca521afaca
        ddq     0x31338431313384313133843131338431
        ddq     0x31338431313384313133843131338431
        ddq     0x6ed41a956ed41a956ed41a956ed41a95
        ddq     0x6ed41a956ed41a956ed41a956ed41a95
        ddq     0x6d4378906d4378906d4378906d437890
        ddq     0x6d4378906d4378906d4378906d437890
        ddq     0xc39c91f2c39c91f2c39c91f2c39c91f2
        ddq     0xc39c91f2c39c91f2c39c91f2c39c91f2
        ddq     0x9eccabbd9eccabbd9eccabbd9eccabbd
        ddq     0x9eccabbd9eccabbd9eccabbd9eccabbd
        ddq     0xb5c9a0e6b5c9a0e6b5c9a0e6b5c9a0e6
        ddq     0xb5c9a0e6b5c9a0e6b5c9a0e6b5c9a0e6
        ddq     0x532fb63c532fb63c532fb63c532fb63c
        ddq     0x532fb63c532fb63c532fb63c532fb63c
        ddq     0xd2c741c6d2c741c6d2c741c6d2c741c6
        ddq     0xd2c741c6d2c741c6d2c741c6d2c741c6
        ddq     0x07237ea307237ea307237ea307237ea3
        ddq     0x07237ea307237ea307237ea307237ea3
        ddq     0xa4954b68a4954b68a4954b68a4954b68
        ddq     0xa4954b68a4954b68a4954b68a4954b68
        ddq     0x4c191d764c191d764c191d764c191d76
        ddq     0x4c191d764c191d764c191d764c191d76


DIGEST_8:
        dd      0x6a09e667, 0x6a09e667, 0x6a09e667, 0x6a09e667
        dd      0x6a09e667, 0x6a09e667, 0x6a09e667, 0x6a09e667
	dd 	0xbb67ae85, 0xbb67ae85, 0xbb67ae85, 0xbb67ae85 
	dd 	0xbb67ae85, 0xbb67ae85, 0xbb67ae85, 0xbb67ae85 
	dd      0x3c6ef372, 0x3c6ef372, 0x3c6ef372, 0x3c6ef372 
	dd      0x3c6ef372, 0x3c6ef372, 0x3c6ef372, 0x3c6ef372 
	dd 	0xa54ff53a, 0xa54ff53a, 0xa54ff53a, 0xa54ff53a 
	dd 	0xa54ff53a, 0xa54ff53a, 0xa54ff53a, 0xa54ff53a 
	dd	0x510e527f, 0x510e527f, 0x510e527f, 0x510e527f
	dd	0x510e527f, 0x510e527f, 0x510e527f, 0x510e527f
	dd 	0x9b05688c, 0x9b05688c, 0x9b05688c, 0x9b05688c 
	dd 	0x9b05688c, 0x9b05688c, 0x9b05688c, 0x9b05688c 
	dd	0x1f83d9ab, 0x1f83d9ab, 0x1f83d9ab, 0x1f83d9ab
	dd	0x1f83d9ab, 0x1f83d9ab, 0x1f83d9ab, 0x1f83d9ab
        dd      0x5be0cd19, 0x5be0cd19, 0x5be0cd19, 0x5be0cd19
        dd      0x5be0cd19, 0x5be0cd19, 0x5be0cd19, 0x5be0cd19


PSHUFFLE_BYTE_FLIP_MASK: 
	dq 0x0405060700010203, 0x0c0d0e0f08090a0b
	dq 0x0405060700010203, 0x0c0d0e0f08090a0b

STACK_ALIGNMENT_MASK:
	dq 0xffffffffffffffe0

section .text

%ifdef WINABI
	%define OUTPUT_PTR	rcx 	; 1st arg
	%define DATA_PTR	rdx 	; 2nd arg
	%define NUM_BLKS 	r8	; 3rd arg
	%define TBL 		rsi
	%define reg1		rdi
%else
	%define OUTPUT_PTR	rdi	; 1st arg
	%define DATA_PTR	rsi	; 2nd arg
	%define NUM_BLKS	rdx	; 3rd arg
	%define TBL 		rcx
	%define reg1 		r8
%endif

%define ROUND	rax

%define inp0 r9
%define inp1 r10
%define inp2 r11
%define inp3 r12
%define inp4 r13
%define inp5 r14
%define inp6 reg1
%define inp7 reg2



; ymm0	a
; ymm1	b
; ymm2	c
; ymm3	d
; ymm4	e
; ymm5	f
; ymm6	g	TMP0
; ymm7	h	TMP1
; ymm8	T1	TT0
; ymm9		TT1
; ymm10		TT2
; ymm11		TT3
; ymm12	a0	TT4
; ymm13	a1	TT5
; ymm14	a2	TT6
; ymm15	TMP	TT7

%define a ymm0
%define b ymm1
%define c ymm2
%define d ymm3
%define e ymm4
%define f ymm5
%define g ymm6
%define h ymm7

%define T1  ymm8

%define a0 ymm12
%define a1 ymm13
%define a2 ymm14
%define TMP ymm15

%define TMP0 ymm6
%define TMP1 ymm7

%define TT0 ymm8
%define TT1 ymm9
%define TT2 ymm10
%define TT3 ymm11
%define TT4 ymm12
%define TT5 ymm13
%define TT6 ymm14
%define TT7 ymm15

%define SHA256_DIGEST_WORD_SIZE  4;
%define SZ8	8*SHA256_DIGEST_WORD_SIZE	; Size of one vector register
%define ROUNDS 64*SZ8

; Define stack usage

;; Assume stack aligned to 32 bytes before call
;; Therefore FRAMESZ mod 32 must be 32-8 = 24
struc stack_frame
  .data		resb	16*SZ8
  .digest	resb	8*SZ8
  .ytmp		resb	4*SZ8
  .regsave	resb    4*64
endstruc
%define FRAMESZ	stack_frame_size
%define _DIGEST	stack_frame.digest
%define _YTMP	stack_frame.ytmp
%define _RSAVE  stack_frame.regsave

%define YTMP0	rsp + _YTMP + 0*SZ8
%define YTMP1	rsp + _YTMP + 1*SZ8
%define YTMP2	rsp + _YTMP + 2*SZ8
%define YTMP3	rsp + _YTMP + 3*SZ8
%define R12 	rsp + _RSAVE + 0*64
%define R13 	rsp + _RSAVE + 1*64
%define R14 	rsp + _RSAVE + 2*64
%define R15 	rsp + _RSAVE + 3*64
	

%define VMOVPS	vmovups


%macro ROTATE_ARGS 0
%xdefine TMP_ h
%xdefine h g
%xdefine g f
%xdefine f e
%xdefine e d
%xdefine d c
%xdefine c b
%xdefine b a
%xdefine a TMP_
%endm

; PRORD reg, imm, tmp
%macro PRORD 3
%define %%reg %1
%define %%imm %2
%define %%tmp %3
	vpslld	%%tmp, %%reg, (32-(%%imm))
	vpsrld	%%reg, %%reg, %%imm
	vpor	%%reg, %%reg, %%tmp
%endmacro

; non-destructive
; PRORD_nd reg, imm, tmp, src
%macro PRORD_nd 4
%define %%reg %1
%define %%imm %2
%define %%tmp %3
%define %%src %4
	;vmovdqa	%%tmp, %%reg
	vpslld	%%tmp, %%src, (32-(%%imm))
	vpsrld	%%reg, %%src, %%imm
	vpor	%%reg, %%reg, %%tmp
%endmacro

; PRORD dst/src, amt
%macro PRORD 2
	PRORD	%1, %2, TMP
%endmacro

; PRORD_nd dst, src, amt
%macro PRORD_nd 3
	PRORD_nd	%1, %3, TMP, %2
%endmacro

;; arguments passed implicitly in preprocessor symbols i, a...h
%macro ROUND_00_15 2
%define %%T1 %1
%define %%i  %2
	PRORD_nd	a0, e, (11-6)	; sig1: a0 = (e >> 5)

	vpxor	a2, f, g	; ch: a2 = f^g
	vpand	a2, a2, e		; ch: a2 = (f^g)&e
	vpxor	a2, a2, g		; a2 = ch

	PRORD_nd	a1, e, 25		; sig1: a1 = (e >> 25)
	vmovdqa	[SZ8*(%%i&0xf) + rsp], %%T1     ; save current temp message
	vpaddd	%%T1, %%T1, [TBL + ROUND]	; T1 = W + K
	vpxor	a0, a0, e	; sig1: a0 = e ^ (e >> 5)
	PRORD	a0, 6		; sig1: a0 = (e >> 6) ^ (e >> 11)
	vpaddd	h, h, a2	; h = h + ch
	PRORD_nd	a2, a, (13-2)	; sig0: a2 = (a >> 11)
	vpaddd	h, h, %%T1	; h = h + ch + W + K
	vpxor	a0, a0, a1	; a0 = sigma1
	PRORD_nd	a1, a, 22	; sig0: a1 = (a >> 22)
	vpxor	%%T1, a, c	; maj: T1 = a^c
	add	ROUND, SZ8	; ROUND++
	vpand	%%T1, %%T1, b	; maj: T1 = (a^c)&b
	vpaddd	h, h, a0

	vpaddd	d, d, h

	vpxor	a2, a2, a	; sig0: a2 = a ^ (a >> 11)
	PRORD	a2, 2		; sig0: a2 = (a >> 2) ^ (a >> 13)
	vpxor	a2, a2, a1	; a2 = sig0
	vpand	a1, a, c	; maj: a1 = a&c
	vpor	a1, a1, %%T1	; a1 = maj
	vpaddd	h, h, a1	; h = h + ch + W + K + maj
	vpaddd	h, h, a2	; h = h + ch + W + K + maj + sigma0

	ROTATE_ARGS
%endm


;; arguments passed implicitly in preprocessor symbols i, a...h
%macro ROUND_16_XX 2
%define %%T1 %1
%define %%i  %2
	vmovdqa	%%T1, [SZ8*((%%i-15)&0xf) + rsp]
	vmovdqa	a1, [SZ8*((%%i-2)&0xf) + rsp]
	vmovdqa	a0, %%T1
	PRORD	%%T1, 18-7
	vmovdqa	a2, a1
	PRORD	a1, 19-17
	vpxor	%%T1, %%T1, a0
	PRORD	%%T1, 7
	vpxor	a1, a1, a2
	PRORD	a1, 17
	vpsrld	a0, a0, 3
	vpxor	%%T1, %%T1, a0
	vpsrld	a2, a2, 10
	vpxor	a1, a1, a2
	vpaddd	%%T1, %%T1, [SZ8*((%%i-16)&0xf) + rsp]   ; + W[i-16]
	vpaddd	a1, a1, [SZ8*((%%i-7)&0xf) + rsp]        ; + W[i-7]
	vpaddd	%%T1, %%T1, a1

	ROUND_00_15 %%T1, %%i

%endm

;; arguments passed implicitly in preprocessor symbols i, a...h
%macro PADDING_ROUND_00_15 1
%define %%T1 %1
	PRORD_nd	a0, e, (11-6)	; sig1: a0 = (e >> 5)

	vpxor	a2, f, g	; ch: a2 = f^g
	vpand	a2, a2, e		; ch: a2 = (f^g)&e
	vpxor	a2, a2, g		; a2 = ch

	PRORD_nd	a1, e, 25		; sig1: a1 = (e >> 25)
	vmovdqa 	%%T1, [TBL + ROUND]	; T1 = W + K
	vpxor	a0, a0, e	; sig1: a0 = e ^ (e >> 5)
	PRORD	a0, 6		; sig1: a0 = (e >> 6) ^ (e >> 11)
	vpaddd	h, h, a2	; h = h + ch
	PRORD_nd	a2, a, (13-2)	; sig0: a2 = (a >> 11)
	vpaddd	h, h, %%T1	; h = h + ch + W + K
	vpxor	a0, a0, a1	; a0 = sigma1
	PRORD_nd	a1, a, 22	; sig0: a1 = (a >> 22)
	vpxor	%%T1, a, c	; maj: T1 = a^c
	add	ROUND, SZ8	; ROUND++
	vpand	%%T1, %%T1, b	; maj: T1 = (a^c)&b
	vpaddd	h, h, a0

	vpaddd	d, d, h

	vpxor	a2, a2, a	; sig0: a2 = a ^ (a >> 11)
	PRORD	a2, 2		; sig0: a2 = (a >> 2) ^ (a >> 13)
	vpxor	a2, a2, a1	; a2 = sig0
	vpand	a1, a, c	; maj: a1 = a&c
	vpor	a1, a1, %%T1	; a1 = maj
	vpaddd	h, h, a1	; h = h + ch + W + K + maj
	vpaddd	h, h, a2	; h = h + ch + W + K + maj + sigma0

	ROTATE_ARGS
%endm



global sha256_8_avx2:function
align 16
sha256_8_avx2:
        endbranch64
	; outer calling routine saves all the XMM registers
	push 	rbp
	mov     rbp,rsp
	and 	rsp, [rel STACK_ALIGNMENT_MASK]
	sub	rsp, FRAMESZ
	mov	[R12], r12
	mov	[R13], r13
	mov	[R14], r14
	mov	[R15], r15
	
.hash_8_blocks:
	cmp 	NUM_BLKS, 8
	jl 	.hash_4_blocks
	xor	ROUND, ROUND

	lea TBL,[rel DIGEST_8]
	vmovdqa	a,[TBL + 0*32]
	vmovdqa	b,[TBL + 1*32]
	vmovdqa	c,[TBL + 2*32]
	vmovdqa	d,[TBL + 3*32]
	vmovdqa	e,[TBL + 4*32]
	vmovdqa	f,[TBL + 5*32]
	vmovdqa	g,[TBL + 6*32]
	vmovdqa	h,[TBL + 7*32]

	lea	TBL,[rel K256_8]
	
%assign i 0
%rep 2
	TRANSPOSE8_U32_LOAD8 TT0, TT1, TT2, TT3, TT4, TT5, TT6, TT7, \
			     DATA_PTR + 0*64, \
			     DATA_PTR + 1*64, \
			     DATA_PTR + 2*64, \
			     DATA_PTR + 3*64, \
			     DATA_PTR + 4*64, \
			     DATA_PTR + 5*64, \
			     DATA_PTR + 6*64, \
			     DATA_PTR + 7*64, \
			     i*32

	vmovdqa	[YTMP0], g
	vmovdqa	[YTMP1], h
	TRANSPOSE8_U32_PRELOADED TT0, TT1, TT2, TT3, TT4, TT5, TT6, TT7,   TMP0, TMP1
	vmovdqa	TMP1, [rel PSHUFFLE_BYTE_FLIP_MASK]
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
	ROUND_00_15	TT0,(i*8+0)
	vmovdqa	TT0, [YTMP0]
	ROUND_00_15	TT1,(i*8+1)
	vmovdqa	TT1, [YTMP1]
	ROUND_00_15	TT2,(i*8+2)
	vmovdqa	TT2, [YTMP2]
	ROUND_00_15	TT3,(i*8+3)
	vmovdqa	TT3, [YTMP3]
	ROUND_00_15	TT0,(i*8+4)
	ROUND_00_15	TT1,(i*8+5)
	ROUND_00_15	TT2,(i*8+6)
	ROUND_00_15	TT3,(i*8+7)
%assign i (i+1)
%endrep

%assign i (i*8)

	jmp	.Lrounds_16_xx
align 16
.Lrounds_16_xx:
%rep 16
	ROUND_16_XX	T1, i
%assign i (i+1)
%endrep

	cmp	ROUND,ROUNDS
	jb	.Lrounds_16_xx

	;; add old digest
	lea TBL,[rel DIGEST_8]
	vpaddd	a, a, [TBL + 0*SZ8]
	vpaddd	b, b, [TBL + 1*SZ8]
	vpaddd	c, c, [TBL + 2*SZ8]
	vpaddd	d, d, [TBL + 3*SZ8]
	vpaddd	e, e, [TBL + 4*SZ8]
	vpaddd	f, f, [TBL + 5*SZ8]
	vpaddd	g, g, [TBL + 6*SZ8]
	vpaddd	h, h, [TBL + 7*SZ8]

	;; rounds with padding

	;; save old digest
	vmovdqa	[rsp + _DIGEST + 0*SZ8], a
	vmovdqa	[rsp + _DIGEST + 1*SZ8], b
	vmovdqa	[rsp + _DIGEST + 2*SZ8], c
	vmovdqa	[rsp + _DIGEST + 3*SZ8], d
	vmovdqa	[rsp + _DIGEST + 4*SZ8], e
	vmovdqa	[rsp + _DIGEST + 5*SZ8], f
	vmovdqa	[rsp + _DIGEST + 6*SZ8], g
	vmovdqa	[rsp + _DIGEST + 7*SZ8], h


	lea TBL,[rel PADDING_8]
	xor ROUND,ROUND
	jmp 	.Lrounds_padding

align 16
.Lrounds_padding:
%rep 64
	PADDING_ROUND_00_15 	T1
%endrep
	;; add old digest
	vpaddd	a, a, [rsp + _DIGEST + 0*SZ8]
	vpaddd	b, b, [rsp + _DIGEST + 1*SZ8]
	vpaddd	c, c, [rsp + _DIGEST + 2*SZ8]
	vpaddd	d, d, [rsp + _DIGEST + 3*SZ8]
	vpaddd	e, e, [rsp + _DIGEST + 4*SZ8]
	vpaddd	f, f, [rsp + _DIGEST + 5*SZ8]
	vpaddd	g, g, [rsp + _DIGEST + 6*SZ8]
	vpaddd	h, h, [rsp + _DIGEST + 7*SZ8]


	;; transpose the digest and convert to little endian to get the registers correctly

	TRANSPOSE8_U32 a, b, c, d, e, f, g, h, TT0, TT1
	vmovdqa	TT0, [rel PSHUFFLE_BYTE_FLIP_MASK]
	vpshufb	a, a, TT0
	vpshufb	b, b, TT0
	vpshufb	c, c, TT0
	vpshufb	d, d, TT0
	vpshufb	e, e, TT0
	vpshufb	f, f, TT0
	vpshufb	g, g, TT0
	vpshufb	h, h, TT0

	;; write to output

	vmovdqu	[OUTPUT_PTR + 0*32],a
	vmovdqu	[OUTPUT_PTR + 1*32],b
	vmovdqu	[OUTPUT_PTR + 2*32],c
	vmovdqu	[OUTPUT_PTR + 3*32],d
	vmovdqu	[OUTPUT_PTR + 4*32],e
	vmovdqu	[OUTPUT_PTR + 5*32],f
	vmovdqu	[OUTPUT_PTR + 6*32],g
	vmovdqu	[OUTPUT_PTR + 7*32],h

	; update pointers and loop

        add 	DATA_PTR, 64*8
	add 	OUTPUT_PTR, 32*8
	sub 	NUM_BLKS, 8

	jmp     .hash_8_blocks

.hash_4_blocks:

	call  	sha256_4_avx

	mov	r12,[R12]
	mov	r13,[R13]
	mov	r14,[R14]
	mov	r15,[R15]

	mov     rsp,rbp
	pop     rbp
	ret

%ifdef LINUX
section .note.GNU-stack noalloc noexec nowrite progbits
%endif
