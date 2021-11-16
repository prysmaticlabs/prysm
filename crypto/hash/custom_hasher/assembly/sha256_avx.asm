;;  sha256_avx.asm
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
;;;
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

;; code to compute quad SHA256 using AVX
;; outer calling routine takes care of save and restore of XMM registers
;; Logic designed/laid out by JDG

;; Stack must be aligned to 16 bytes before call
;; Windows clobbers:  rax rdx             r8 r9 r10 r11 
;; Windows preserves:         rcx     rsi rdi rbp                r12 r13 r14 r15
;;
;; Linux clobbers:    rax rsi         r8 r9 r10 r11 
;; Linux preserves:           rcx rdx     rdi rbp                r12 r13 r14 r15
;;
;; clobbers xmm0-15


extern sha256_1_avx

%ifdef WINABI
	%define OUTPUT_PTR	rcx 	; 1st arg
	%define DATA_PTR	rdx 	; 2nd arg
	%define NUM_BLKS 	r8	; 3rd arg
	%define TBL 		rsi
%else
	%define OUTPUT_PTR	rdi	; 1st arg
	%define DATA_PTR	rsi	; 2nd arg
	%define NUM_BLKS	rdx	; 3rd arg
	%define TBL 		rcx
%endif

%define ROUND	rbx

%define inp0 r8
%define inp1 r9
%define inp2 r10
%define inp3 r11

%define a xmm0
%define b xmm1
%define c xmm2
%define d xmm3
%define e xmm4
%define f xmm5
%define g xmm6
%define h xmm7

%define a0 xmm8
%define a1 xmm9
%define a2 xmm10

%define TT0 xmm14
%define TT1 xmm13
%define TT2 xmm12
%define TT3 xmm11
%define TT4 xmm10
%define TT5 xmm9

%define T1  xmm14
%define TMP xmm15

%define SHA256_DIGEST_WORD_SIZE  4
%define NUM_SHA256_DIGEST_WORDS 8
%define SZ4	4*SHA256_DIGEST_WORD_SIZE	; Size of one vector register
%define ROUNDS 64*SZ4

; Define stack usage
struc STACK
	_DATA:		resb	SZ4 * 16
	_DIGEST:	resb	SZ4 * NUM_SHA256_DIGEST_WORDS
	_RBX: 		resb	8
			resb 	16 	
endstruc

%define VMOVPS	vmovups

; transpose r0, r1, r2, r3, t0, t1
; "transpose" data in {r0..r3} using temps {t0,t1}
; Input looks like: {r0 r1 r2 r3}
; r0 = {a3 a2 a1 a0}
; r1 = {b3 b2 b1 b0}
; r2 = {c3 c2 c1 c0}
; r3 = {d3 d2 d1 d0}
;
; output looks like: {t0 r1 r0 r3}
; t0 = {d0 c0 b0 a0}
; r1 = {d1 c1 b1 a1}
; r0 = {d2 c2 b2 a2}
; r3 = {d3 c3 b3 a3}
;
%macro TRANSPOSE 6
%define %%r0 %1
%define %%r1 %2
%define %%r2 %3
%define %%r3 %4
%define %%t0 %5
%define %%t1 %6
	vshufps	%%t0, %%r0, %%r1, 0x44	; t0 = {b1 b0 a1 a0}
	vshufps	%%r0, %%r0, %%r1, 0xEE	; r0 = {b3 b2 a3 a2}

	vshufps	%%t1, %%r2, %%r3, 0x44	; t1 = {d1 d0 c1 c0}
	vshufps	%%r2, %%r2, %%r3, 0xEE	; r2 = {d3 d2 c3 c2}

	vshufps	%%r1, %%t0, %%t1, 0xDD	; r1 = {d1 c1 b1 a1}

	vshufps	%%r3, %%r0, %%r2, 0xDD	; r3 = {d3 c3 b3 a3}

	vshufps	%%r0, %%r0, %%r2, 0x88	; r0 = {d2 c2 b2 a2}
	vshufps	%%t0, %%t0, %%t1, 0x88	; t0 = {d0 c0 b0 a0}
%endmacro



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
	vpand	a2, a2, e	; ch: a2 = (f^g)&e
	vpxor	a2, a2, g	; a2 = ch

	PRORD_nd	a1, e, 25		; sig1: a1 = (e >> 25)
	vmovdqa	[SZ4*(%%i&0xf) + rsp + _DATA], %%T1
	vpaddd	%%T1, %%T1, [TBL + ROUND]	; T1 = W + K
	vpxor	a0, a0, e	; sig1: a0 = e ^ (e >> 5)
	PRORD	a0, 6		; sig1: a0 = (e >> 6) ^ (e >> 11)
	vpaddd	h, h, a2	; h = h + ch
	PRORD_nd	a2, a, (13-2)	; sig0: a2 = (a >> 11)
	vpaddd	h, h, %%T1	; h = h + ch + W + K
	vpxor	a0, a0, a1	; a0 = sigma1
	PRORD_nd	a1, a, 22	; sig0: a1 = (a >> 22)
	vpxor	%%T1, a, c	; maj: T1 = a^c
	add	ROUND, SZ4	; ROUND++
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
	vmovdqa	%%T1, [SZ4*((%%i-15)&0xf) + rsp + _DATA]
	vmovdqa	a1, [SZ4*((%%i-2)&0xf) + rsp + _DATA]
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
	vpaddd	%%T1, %%T1, [SZ4*((%%i-16)&0xf) + rsp + _DATA]
	vpaddd	a1, a1, [SZ4*((%%i-7)&0xf) + rsp + _DATA]
	vpaddd	%%T1, %%T1, a1

	ROUND_00_15 %%T1, %%i
%endm

;; arguments passed implicitly in preprocessor symbols i, a...h
%macro PADDING_ROUND_00_15 1
%define %%T1 %1
	PRORD_nd	a0, e, (11-6)	; sig1: a0 = (e >> 5)

	vpxor	a2, f, g	; ch: a2 = f^g
	vpand	a2, a2, e	; ch: a2 = (f^g)&e
	vpxor	a2, a2, g	; a2 = ch

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
	add	ROUND, SZ4	; ROUND++
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



section .data
default rel
align 64

K256_4:
	dq	0x428a2f98428a2f98, 0x428a2f98428a2f98
	dq	0x7137449171374491, 0x7137449171374491
	dq	0xb5c0fbcfb5c0fbcf, 0xb5c0fbcfb5c0fbcf
	dq	0xe9b5dba5e9b5dba5, 0xe9b5dba5e9b5dba5
	dq	0x3956c25b3956c25b, 0x3956c25b3956c25b
	dq	0x59f111f159f111f1, 0x59f111f159f111f1
	dq	0x923f82a4923f82a4, 0x923f82a4923f82a4
	dq	0xab1c5ed5ab1c5ed5, 0xab1c5ed5ab1c5ed5
	dq	0xd807aa98d807aa98, 0xd807aa98d807aa98
	dq	0x12835b0112835b01, 0x12835b0112835b01
	dq	0x243185be243185be, 0x243185be243185be
	dq	0x550c7dc3550c7dc3, 0x550c7dc3550c7dc3
	dq	0x72be5d7472be5d74, 0x72be5d7472be5d74
	dq	0x80deb1fe80deb1fe, 0x80deb1fe80deb1fe
	dq	0x9bdc06a79bdc06a7, 0x9bdc06a79bdc06a7
	dq	0xc19bf174c19bf174, 0xc19bf174c19bf174
	dq	0xe49b69c1e49b69c1, 0xe49b69c1e49b69c1
	dq	0xefbe4786efbe4786, 0xefbe4786efbe4786
	dq	0x0fc19dc60fc19dc6, 0x0fc19dc60fc19dc6
	dq	0x240ca1cc240ca1cc, 0x240ca1cc240ca1cc
	dq	0x2de92c6f2de92c6f, 0x2de92c6f2de92c6f
	dq	0x4a7484aa4a7484aa, 0x4a7484aa4a7484aa
	dq	0x5cb0a9dc5cb0a9dc, 0x5cb0a9dc5cb0a9dc
	dq	0x76f988da76f988da, 0x76f988da76f988da
	dq	0x983e5152983e5152, 0x983e5152983e5152
	dq	0xa831c66da831c66d, 0xa831c66da831c66d
	dq	0xb00327c8b00327c8, 0xb00327c8b00327c8
	dq	0xbf597fc7bf597fc7, 0xbf597fc7bf597fc7
	dq	0xc6e00bf3c6e00bf3, 0xc6e00bf3c6e00bf3
	dq	0xd5a79147d5a79147, 0xd5a79147d5a79147
	dq	0x06ca635106ca6351, 0x06ca635106ca6351
	dq	0x1429296714292967, 0x1429296714292967
	dq	0x27b70a8527b70a85, 0x27b70a8527b70a85
	dq	0x2e1b21382e1b2138, 0x2e1b21382e1b2138
	dq	0x4d2c6dfc4d2c6dfc, 0x4d2c6dfc4d2c6dfc
	dq	0x53380d1353380d13, 0x53380d1353380d13
	dq	0x650a7354650a7354, 0x650a7354650a7354
	dq	0x766a0abb766a0abb, 0x766a0abb766a0abb
	dq	0x81c2c92e81c2c92e, 0x81c2c92e81c2c92e
	dq	0x92722c8592722c85, 0x92722c8592722c85
	dq	0xa2bfe8a1a2bfe8a1, 0xa2bfe8a1a2bfe8a1
	dq	0xa81a664ba81a664b, 0xa81a664ba81a664b
	dq	0xc24b8b70c24b8b70, 0xc24b8b70c24b8b70
	dq	0xc76c51a3c76c51a3, 0xc76c51a3c76c51a3
	dq	0xd192e819d192e819, 0xd192e819d192e819
	dq	0xd6990624d6990624, 0xd6990624d6990624
	dq	0xf40e3585f40e3585, 0xf40e3585f40e3585
	dq	0x106aa070106aa070, 0x106aa070106aa070
	dq	0x19a4c11619a4c116, 0x19a4c11619a4c116
	dq	0x1e376c081e376c08, 0x1e376c081e376c08
	dq	0x2748774c2748774c, 0x2748774c2748774c
	dq	0x34b0bcb534b0bcb5, 0x34b0bcb534b0bcb5
	dq	0x391c0cb3391c0cb3, 0x391c0cb3391c0cb3
	dq	0x4ed8aa4a4ed8aa4a, 0x4ed8aa4a4ed8aa4a
	dq	0x5b9cca4f5b9cca4f, 0x5b9cca4f5b9cca4f
	dq	0x682e6ff3682e6ff3, 0x682e6ff3682e6ff3
	dq	0x748f82ee748f82ee, 0x748f82ee748f82ee
	dq	0x78a5636f78a5636f, 0x78a5636f78a5636f
	dq	0x84c8781484c87814, 0x84c8781484c87814
	dq	0x8cc702088cc70208, 0x8cc702088cc70208
	dq	0x90befffa90befffa, 0x90befffa90befffa
	dq	0xa4506ceba4506ceb, 0xa4506ceba4506ceb
	dq	0xbef9a3f7bef9a3f7, 0xbef9a3f7bef9a3f7
	dq	0xc67178f2c67178f2, 0xc67178f2c67178f2

PADDING_4:
        dq      0xc28a2f98c28a2f98, 0xc28a2f98c28a2f98
        dq      0x7137449171374491, 0x7137449171374491
        dq      0xb5c0fbcfb5c0fbcf, 0xb5c0fbcfb5c0fbcf
        dq      0xe9b5dba5e9b5dba5, 0xe9b5dba5e9b5dba5
        dq      0x3956c25b3956c25b, 0x3956c25b3956c25b
        dq      0x59f111f159f111f1, 0x59f111f159f111f1
        dq      0x923f82a4923f82a4, 0x923f82a4923f82a4
        dq      0xab1c5ed5ab1c5ed5, 0xab1c5ed5ab1c5ed5
        dq      0xd807aa98d807aa98, 0xd807aa98d807aa98
        dq      0x12835b0112835b01, 0x12835b0112835b01
        dq      0x243185be243185be, 0x243185be243185be
        dq      0x550c7dc3550c7dc3, 0x550c7dc3550c7dc3
        dq      0x72be5d7472be5d74, 0x72be5d7472be5d74
        dq      0x80deb1fe80deb1fe, 0x80deb1fe80deb1fe
        dq      0x9bdc06a79bdc06a7, 0x9bdc06a79bdc06a7
        dq      0xc19bf374c19bf374, 0xc19bf374c19bf374
        dq      0x649b69c1649b69c1, 0x649b69c1649b69c1
        dq      0xf0fe4786f0fe4786, 0xf0fe4786f0fe4786
        dq      0x0fe1edc60fe1edc6, 0x0fe1edc60fe1edc6
        dq      0x240cf254240cf254, 0x240cf254240cf254
        dq      0x4fe9346f4fe9346f, 0x4fe9346f4fe9346f
        dq      0x6cc984be6cc984be, 0x6cc984be6cc984be
        dq      0x61b9411e61b9411e, 0x61b9411e61b9411e
        dq      0x16f988fa16f988fa, 0x16f988fa16f988fa
        dq      0xf2c65152f2c65152, 0xf2c65152f2c65152
        dq      0xa88e5a6da88e5a6d, 0xa88e5a6da88e5a6d
        dq      0xb019fc65b019fc65, 0xb019fc65b019fc65
        dq      0xb9d99ec7b9d99ec7, 0xb9d99ec7b9d99ec7
        dq      0x9a1231c39a1231c3, 0x9a1231c39a1231c3
        dq      0xe70eeaa0e70eeaa0, 0xe70eeaa0e70eeaa0
        dq      0xfdb1232bfdb1232b, 0xfdb1232bfdb1232b
        dq      0xc7353eb0c7353eb0, 0xc7353eb0c7353eb0
        dq      0x3069bad53069bad5, 0x3069bad53069bad5
        dq      0xcb976d5fcb976d5f, 0xcb976d5fcb976d5f
        dq      0x5a0f118f5a0f118f, 0x5a0f118f5a0f118f
        dq      0xdc1eeefddc1eeefd, 0xdc1eeefddc1eeefd
        dq      0x0a35b6890a35b689, 0x0a35b6890a35b689
        dq      0xde0b7a04de0b7a04, 0xde0b7a04de0b7a04
        dq      0x58f4ca9d58f4ca9d, 0x58f4ca9d58f4ca9d
        dq      0xe15d5b16e15d5b16, 0xe15d5b16e15d5b16
        dq      0x007f3e86007f3e86, 0x007f3e86007f3e86
        dq      0x3708898037088980, 0x3708898037088980
        dq      0xa507ea32a507ea32, 0xa507ea32a507ea32
        dq      0x6fab95376fab9537, 0x6fab95376fab9537
        dq      0x1740611017406110, 0x1740611017406110
        dq      0x0d8cd6f10d8cd6f1, 0x0d8cd6f10d8cd6f1
        dq      0xcdaa3b6dcdaa3b6d, 0xcdaa3b6dcdaa3b6d
        dq      0xc0bbbe37c0bbbe37, 0xc0bbbe37c0bbbe37
        dq      0x83613bda83613bda, 0x83613bda83613bda
        dq      0xdb48a363db48a363, 0xdb48a363db48a363
        dq      0x0b02e9310b02e931, 0x0b02e9310b02e931
        dq      0x6fd15ca76fd15ca7, 0x6fd15ca76fd15ca7
        dq      0x521afaca521afaca, 0x521afaca521afaca
        dq      0x3133843131338431, 0x3133843131338431
        dq      0x6ed41a956ed41a95, 0x6ed41a956ed41a95
        dq      0x6d4378906d437890, 0x6d4378906d437890
        dq      0xc39c91f2c39c91f2, 0xc39c91f2c39c91f2
        dq      0x9eccabbd9eccabbd, 0x9eccabbd9eccabbd
        dq      0xb5c9a0e6b5c9a0e6, 0xb5c9a0e6b5c9a0e6
        dq      0x532fb63c532fb63c, 0x532fb63c532fb63c
        dq      0xd2c741c6d2c741c6, 0xd2c741c6d2c741c6
        dq      0x07237ea307237ea3, 0x07237ea307237ea3
        dq      0xa4954b68a4954b68, 0xa4954b68a4954b68
        dq      0x4c191d764c191d76, 0x4c191d764c191d76

DIGEST_4:
        dd      0x6a09e667, 0x6a09e667, 0x6a09e667, 0x6a09e667
	dd 	0xbb67ae85, 0xbb67ae85, 0xbb67ae85, 0xbb67ae85 
	dd      0x3c6ef372, 0x3c6ef372, 0x3c6ef372, 0x3c6ef372 
	dd 	0xa54ff53a, 0xa54ff53a, 0xa54ff53a, 0xa54ff53a 
	dd	0x510e527f, 0x510e527f, 0x510e527f, 0x510e527f
	dd 	0x9b05688c, 0x9b05688c, 0x9b05688c, 0x9b05688c 
	dd	0x1f83d9ab, 0x1f83d9ab, 0x1f83d9ab, 0x1f83d9ab
        dd      0x5be0cd19, 0x5be0cd19, 0x5be0cd19, 0x5be0cd19

PSHUFFLE_BYTE_FLIP_MASK: 
	dq 0x0405060700010203, 0x0c0d0e0f08090a0b

section .text

global sha256_4_avx:function
align 16
sha256_4_avx:
        endbranch64
	; outer calling routine saves all the XMM registers
	sub	rsp, STACK_size
	mov     [rsp + _RBX],rbx

.hash_4_blocks:
	cmp 	NUM_BLKS, 4
	jl 	.hash_1_block

	xor	ROUND, ROUND

	;; Load the pre-transposed incoming digest.
	lea TBL,[rel DIGEST_4]
	vmovdqa	a,[TBL + 0*SZ4]
	vmovdqa	b,[TBL + 1*SZ4]
	vmovdqa	c,[TBL + 2*SZ4]
	vmovdqa	d,[TBL + 3*SZ4]
	vmovdqa	e,[TBL + 4*SZ4]
	vmovdqa	f,[TBL + 5*SZ4]
	vmovdqa	g,[TBL + 6*SZ4]
	vmovdqa	h,[TBL + 7*SZ4]

	lea	TBL,[rel K256_4]

%assign i 0
%rep 4
	vmovdqa	TMP, [rel PSHUFFLE_BYTE_FLIP_MASK]
	VMOVPS	TT2,[DATA_PTR + 0*64 + i*16]
	VMOVPS	TT1,[DATA_PTR + 1*64 + i*16]
	VMOVPS	TT4,[DATA_PTR + 2*64 + i*16]
	VMOVPS	TT3,[DATA_PTR + 3*64 + i*16]
	TRANSPOSE	TT2, TT1, TT4, TT3, TT0, TT5
	vpshufb	TT0, TT0, TMP
	vpshufb	TT1, TT1, TMP
	vpshufb	TT2, TT2, TMP
	vpshufb	TT3, TT3, TMP
	ROUND_00_15	TT0,(i*4+0)
	ROUND_00_15	TT1,(i*4+1)
	ROUND_00_15	TT2,(i*4+2)
	ROUND_00_15	TT3,(i*4+3)
%assign i (i+1)
%endrep

%assign i (i*4)

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
	lea TBL,[rel DIGEST_4]
	vpaddd	a, a, [TBL + 0*SZ4]
	vpaddd	b, b, [TBL + 1*SZ4]
	vpaddd	c, c, [TBL + 2*SZ4]
	vpaddd	d, d, [TBL + 3*SZ4]
	vpaddd	e, e, [TBL + 4*SZ4]
	vpaddd	f, f, [TBL + 5*SZ4]
	vpaddd	g, g, [TBL + 6*SZ4]
	vpaddd	h, h, [TBL + 7*SZ4]

        ;; rounds with padding
        
        ;; save old digest
        
	vmovdqa	[rsp + _DIGEST + 0*SZ4], a
	vmovdqa	[rsp + _DIGEST + 1*SZ4], b
	vmovdqa	[rsp + _DIGEST + 2*SZ4], c
	vmovdqa	[rsp + _DIGEST + 3*SZ4], d
	vmovdqa	[rsp + _DIGEST + 4*SZ4], e
	vmovdqa	[rsp + _DIGEST + 5*SZ4], f
	vmovdqa	[rsp + _DIGEST + 6*SZ4], g
	vmovdqa	[rsp + _DIGEST + 7*SZ4], h

        lea   TBL,[rel PADDING_4]
        xor   ROUND,ROUND
        jmp   .Lrounds_padding

align 16
.Lrounds_padding:
%rep 64
	PADDING_ROUND_00_15 	T1
%endrep
	;; add old digest
	vpaddd	a, a, [rsp + _DIGEST + 0*SZ4]
	vpaddd	b, b, [rsp + _DIGEST + 1*SZ4]
	vpaddd	c, c, [rsp + _DIGEST + 2*SZ4]
	vpaddd	d, d, [rsp + _DIGEST + 3*SZ4]
	vpaddd	e, e, [rsp + _DIGEST + 4*SZ4]
	vpaddd	f, f, [rsp + _DIGEST + 5*SZ4]
	vpaddd	g, g, [rsp + _DIGEST + 6*SZ4]
	vpaddd	h, h, [rsp + _DIGEST + 7*SZ4]

	;; transpose the digest and convert to little endian to get the registers correctly

	TRANSPOSE a, b, c, d, TT0, TT1
        TRANSPOSE e, f, g, h, TT2, TT1

	vmovdqa	TMP, [rel PSHUFFLE_BYTE_FLIP_MASK]
        vpshufb TT0, TMP
        vpshufb TT2, TMP
        vpshufb b, TMP
        vpshufb f, TMP
        vpshufb a, TMP
        vpshufb e, TMP
        vpshufb d, TMP
        vpshufb h, TMP


	;; write to output

	vmovdqu	[OUTPUT_PTR + 0*SZ4],TT0
	vmovdqu	[OUTPUT_PTR + 1*SZ4],TT2
	vmovdqu	[OUTPUT_PTR + 2*SZ4],b
	vmovdqu	[OUTPUT_PTR + 3*SZ4],f
	vmovdqu	[OUTPUT_PTR + 4*SZ4],a
	vmovdqu	[OUTPUT_PTR + 5*SZ4],e
	vmovdqu	[OUTPUT_PTR + 6*SZ4],d
	vmovdqu	[OUTPUT_PTR + 7*SZ4],h

	; update pointers and loop

        add 	DATA_PTR, 64*4
	add 	OUTPUT_PTR, 32*4
	sub 	NUM_BLKS, 4
        jmp     .hash_4_blocks

.hash_1_block:
        test     NUM_BLKS,NUM_BLKS
        jz      .done_hash
        call    sha256_1_avx
        add     DATA_PTR, 64
        add     OUTPUT_PTR, 32
        dec     NUM_BLKS
        jmp     .hash_1_block

.done_hash:
	mov     rbx,[rsp + _RBX]
	add	rsp, STACK_size
	ret

%ifdef LINUX
section .note.GNU-stack noalloc noexec nowrite progbits
%endif
