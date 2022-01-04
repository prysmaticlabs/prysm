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
.LK256:
	.long	0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5
	.long	0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5
	.long	0xd807aa98,0x12835b01,0x243185be,0x550c7dc3
	.long	0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174
	.long	0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc
	.long	0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da
	.long	0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7
	.long	0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967
	.long	0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13
	.long	0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85
	.long	0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3
	.long	0xd192e819,0xd6990624,0xf40e3585,0x106aa070
	.long	0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5
	.long	0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3
	.long	0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208
	.long	0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2

.LDIGEST:
        .long   0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a
	.long	0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19

.LPADDING:
        .long      0xc28a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5
        .long      0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5
        .long      0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3
        .long      0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf374
        .long      0x649b69c1, 0xf0fe4786, 0x0fe1edc6, 0x240cf254
        .long      0x4fe9346f, 0x6cc984be, 0x61b9411e, 0x16f988fa
        .long      0xf2c65152, 0xa88e5a6d, 0xb019fc65, 0xb9d99ec7
        .long      0x9a1231c3, 0xe70eeaa0, 0xfdb1232b, 0xc7353eb0
        .long      0x3069bad5, 0xcb976d5f, 0x5a0f118f, 0xdc1eeefd
        .long      0x0a35b689, 0xde0b7a04, 0x58f4ca9d, 0xe15d5b16
        .long      0x007f3e86, 0x37088980, 0xa507ea32, 0x6fab9537
        .long      0x17406110, 0x0d8cd6f1, 0xcdaa3b6d, 0xc0bbbe37
        .long      0x83613bda, 0xdb48a363, 0x0b02e931, 0x6fd15ca7
        .long      0x521afaca, 0x31338431, 0x6ed41a95, 0x6d437890
        .long      0xc39c91f2, 0x9eccabbd, 0xb5c9a0e6, 0x532fb63c
        .long      0xd2c741c6, 0x07237ea3, 0xa4954b68, 0x4c191d76


.LPSHUFFLE_BYTE_FLIP_MASK: //.longq 0x0c0d0e0f08090a0b0405060700010203
	.quad 0x0405060700010203, 0x0c0d0e0f08090a0b

// shuffle xBxA -> 00BA
.L_SHUF_00BA:              //d.quad 0xFFFFFFFFFFFFFFFF0b0a090803020100
	.quad 0x0b0a090803020100, 0xFFFFFFFFFFFFFFFF

// shuffle xDxC -> DC00
.L_SHUF_DC00:              //d.quad 0x0b0a090803020100FFFFFFFFFFFFFFFF
	.quad 0xFFFFFFFFFFFFFFFF, 0x0b0a090803020100

.text
# define VMOVDQ vmovdqu

.macro MY_ROR src, shf
	shld        \src, \src, (32-\shf)
.endm

// COPY_XMM_AND_BSWAP xmm, [mem], byte_flip_mask
// Load xmm with mem and byte swap each dword
.macro COPY_XMM_AND_BSWAP dst, src, msk
	VMOVDQ      \dst, \src
	vpshufb     \dst, \dst, \msk
.endm

.eqv X0, xmm4
.eqv X1, xmm5
.eqv X2, xmm6
.eqv X3, xmm7

.equiv XTMP0, xmm0
.equiv XTMP1, xmm1
.equiv XTMP2, xmm2
.equiv XTMP3, xmm3
.equiv XTMP4, xmm8
.equiv XFER, xmm9
.equiv XTMP5, xmm11

.equiv SHUF_00BA,	xmm10 // shuffle xBxA -> 00BA
.equiv SHUF_DC00,	xmm12 // shuffle xDxC -> DC00
.equiv BYTE_FLIP_MASK,	xmm13

#ifdef __WIN64__
                    .eqv OUTPUT_PTR, rcx            // 1st arg
                    .eqv DATA_PTR, rdx              // 2nd arg
                    .eqv d_, ebp
                    .equiv count, r8                // 3rd arg

                    .eqv TBL, rsi
                    .eqv c_, edi
                    #define _XMM_SAVE 64
                    #define stack_size 192
#else
                    .eqv OUTPUT_PTR, rdi            // 1st arg
                    .eqv DATA_PTR, rsi              // 2nd arg
                    .eqv c_, ebp
                    .equiv count, rdx               // 3rd arg

                    .eqv TBL, rcx
                    .eqv d_, r8d
                    #define stack_size 88
#endif

#define _DIGEST 32

.eqv a_, eax
.eqv b_, ebx
.eqv e_, r9d
.eqv f_, r10d
.eqv g_, r11d
.eqv h_, r12d

.equiv y0, r13d
.equiv y1, r14d
.equiv y2, r15d


.macro rotate_Xs
.eqv X_, X0
.eqv X0, X1
.eqv X1, X2
.eqv X2, X3
.eqv X3, X_
.endm

.macro ROTATE_ARGS
.eqv TMP_, h_
.eqv h_, g_
.eqv g_, f_
.eqv f_, e_
.eqv e_, d_
.eqv d_, c_
.eqv c_, b_
.eqv b_, a_
.eqv a_, TMP_
.endm

.macro FOUR_ROUNDS_AND_SCHED
		//// compute s0 four at a time and s1 two at a time
		//// compute W[-16] + W[-7] 4 at a time
		//vmovdqa	XTMP0, X3
	mov	y0, e_		// y0 = e
	MY_ROR	y0, (25-11)	// y0 = e >> (25-11)
	mov	y1, a_		// y1 = a
		vpalignr	XTMP0, X3, X2, 4	// XTMP0 = W[-7]
	MY_ROR	y1, (22-13)	// y1 = a >> (22-13)
	xor	y0, e_		// y0 = e ^ (e >> (25-11))
	mov	y2, f_		// y2 = f
	MY_ROR	y0, (11-6)	// y0 = (e >> (11-6)) ^ (e >> (25-6))
		//vmovdqa	XTMP1, X1
	xor	y1, a_		// y1 = a ^ (a >> (22-13)
	xor	y2, g_		// y2 = f^g
		vpaddd	XTMP0, XTMP0, X0	// XTMP0 = W[-7] + W[-16]
	xor	y0, e_		// y0 = e ^ (e >> (11-6)) ^ (e >> (25-6))
	and	y2, e_		// y2 = (f^g)&e
	MY_ROR	y1, (13-2)	// y1 = (a >> (13-2)) ^ (a >> (22-2))
		//// compute s0
		vpalignr	XTMP1, X1, X0, 4	// XTMP1 = W[-15]
	xor	y1, a_		// y1 = a ^ (a >> (13-2)) ^ (a >> (22-2))
	MY_ROR	y0, 6		// y0 = S1 = (e>>6) & (e>>11) ^ (e>>25)
	xor	y2, g_		// y2 = CH = ((f^g)&e)^g

	MY_ROR	y1, 2		// y1 = S0 = (a>>2) ^ (a>>13) ^ (a>>22)
	add	y2, y0		// y2 = S1 + CH
	add	y2, [rsp + 0*4]	// y2 = k + w + S1 + CH

	mov	y0, a_		// y0 = a
	add	h_, y2		// h = h + S1 + CH + k + w
	mov	y2, a_		// y2 = a

		vpsrld	XTMP2, XTMP1, 7

	or	y0, c_		// y0 = a|c
	add	d_, h_		// d = d + h + S1 + CH + k + w
	and	y2, c_		// y2 = a&c

		vpslld	XTMP3, XTMP1, (32-7)

	and	y0, b_		// y0 = (a|c)&b
	add	h_, y1		// h = h + S1 + CH + k + w + S0

		vpor	XTMP3, XTMP3, XTMP2	// XTMP1 = W[-15] MY_ROR 7

	or	y0, y2		// y0 = MAJ = (a|c)&b)|(a&c)
	add	h_, y0		// h = h + S1 + CH + k + w + S0 + MAJ

ROTATE_ARGS

	mov	y0, e_		// y0 = e
	mov	y1, a_		// y1 = a


	MY_ROR	y0, (25-11)	// y0 = e >> (25-11)
	xor	y0, e_		// y0 = e ^ (e >> (25-11))
	mov	y2, f_		// y2 = f
	MY_ROR	y1, (22-13)	// y1 = a >> (22-13)

		vpsrld	XTMP2, XTMP1,18

	xor	y1, a_		// y1 = a ^ (a >> (22-13)
	MY_ROR	y0, (11-6)	// y0 = (e >> (11-6)) ^ (e >> (25-6))
	xor	y2, g_		// y2 = f^g

		vpsrld	XTMP4, XTMP1, 3	// XTMP4 = W[-15] >> 3

	MY_ROR	y1, (13-2)	// y1 = (a >> (13-2)) ^ (a >> (22-2))
	xor	y0, e_		// y0 = e ^ (e >> (11-6)) ^ (e >> (25-6))
	and	y2, e_		// y2 = (f^g)&e
	MY_ROR	y0, 6		// y0 = S1 = (e>>6) & (e>>11) ^ (e>>25)

		vpslld	XTMP1, XTMP1, (32-18)

	xor	y1, a_		// y1 = a ^ (a >> (13-2)) ^ (a >> (22-2))
	xor	y2, g_		// y2 = CH = ((f^g)&e)^g

		vpxor	XTMP3, XTMP3, XTMP1

	add	y2, y0		// y2 = S1 + CH
	add	y2, [rsp + 1*4]	// y2 = k + w + S1 + CH
	MY_ROR	y1, 2		// y1 = S0 = (a>>2) ^ (a>>13) ^ (a>>22)

		vpxor	XTMP3, XTMP3, XTMP2	// XTMP1 = W[-15] MY_ROR 7 ^ W[-15] MY_ROR 18

	mov	y0, a_		// y0 = a
	add	h_, y2		// h = h + S1 + CH + k + w
	mov	y2, a_		// y2 = a

		vpxor	XTMP1, XTMP3, XTMP4	// XTMP1 = s0

	or	y0, c_		// y0 = a|c
	add	d_, h_		// d = d + h + S1 + CH + k + w
	and	y2, c_		// y2 = a&c
		//// compute low s1
		vpshufd	XTMP2, X3, 0b11111010	// XTMP2 = W[-2] {BBAA}
	and	y0, b_		// y0 = (a|c)&b
	add	h_, y1		// h = h + S1 + CH + k + w + S0
		vpaddd	XTMP0, XTMP0, XTMP1	// XTMP0 = W[-16] + W[-7] + s0
	or	y0, y2		// y0 = MAJ = (a|c)&b)|(a&c)
	add	h_, y0		// h = h + S1 + CH + k + w + S0 + MAJ

ROTATE_ARGS
		//vmovdqa	XTMP3, XTMP2	// XTMP3 = W[-2] {BBAA}

	mov	y0, e_		// y0 = e
	mov	y1, a_		// y1 = a
	MY_ROR	y0, (25-11)	// y0 = e >> (25-11)

		//vmovdqa	XTMP4, XTMP2	// XTMP4 = W[-2] {BBAA}

	xor	y0, e_		// y0 = e ^ (e >> (25-11))
	MY_ROR	y1, (22-13)	// y1 = a >> (22-13)
	mov	y2, f_		// y2 = f
	xor	y1, a_		// y1 = a ^ (a >> (22-13)
	MY_ROR	y0, (11-6)	// y0 = (e >> (11-6)) ^ (e >> (25-6))

		vpsrld	XTMP4, XTMP2, 10	// XTMP4 = W[-2] >> 10 {BBAA}

	xor	y2, g_		// y2 = f^g

		vpsrlq	XTMP3, XTMP2, 19	// XTMP3 = W[-2] MY_ROR 19 {xBxA}

	xor	y0, e_		// y0 = e ^ (e >> (11-6)) ^ (e >> (25-6))
	and	y2, e_		// y2 = (f^g)&e

		vpsrlq	XTMP2, XTMP2, 17	// XTMP2 = W[-2] MY_ROR 17 {xBxA}

	MY_ROR	y1, (13-2)	// y1 = (a >> (13-2)) ^ (a >> (22-2))
	xor	y1, a_		// y1 = a ^ (a >> (13-2)) ^ (a >> (22-2))
	xor	y2, g_		// y2 = CH = ((f^g)&e)^g
	MY_ROR	y0, 6		// y0 = S1 = (e>>6) & (e>>11) ^ (e>>25)
		vpxor	XTMP2, XTMP2, XTMP3
	add	y2, y0		// y2 = S1 + CH
	MY_ROR	y1, 2		// y1 = S0 = (a>>2) ^ (a>>13) ^ (a>>22)
	add	y2, [rsp + 2*4]	// y2 = k + w + S1 + CH
		vpxor	XTMP4, XTMP4, XTMP2	// XTMP4 = s1 {xBxA}
	mov	y0, a_		// y0 = a
	add	h_, y2		// h = h + S1 + CH + k + w
	mov	y2, a_		// y2 = a
		vpshufb	XTMP4, XTMP4, SHUF_00BA	// XTMP4 = s1 {00BA}
	or	y0, c_		// y0 = a|c
	add	d_, h_		// d = d + h + S1 + CH + k + w
	and	y2, c_		// y2 = a&c
		vpaddd	XTMP0, XTMP0, XTMP4	// XTMP0 = {..., ..., W[1], W[0]}
	and	y0, b_		// y0 = (a|c)&b
	add	h_, y1		// h = h + S1 + CH + k + w + S0
		//// compute high s1
		vpshufd	XTMP2, XTMP0, 0b01010000	// XTMP2 = W[-2] {DDCC}
	or	y0, y2		// y0 = MAJ = (a|c)&b)|(a&c)
	add	h_, y0		// h = h + S1 + CH + k + w + S0 + MAJ

ROTATE_ARGS
		//vmovdqa	XTMP3, XTMP2	// XTMP3 = W[-2] {DDCC}
	mov	y0, e_		// y0 = e
	MY_ROR	y0, (25-11)	// y0 = e >> (25-11)
	mov	y1, a_		// y1 = a
		//vmovdqa	XTMP5,    XTMP2	// XTMP5    = W[-2] {DDCC}
	MY_ROR	y1, (22-13)	// y1 = a >> (22-13)
	xor	y0, e_		// y0 = e ^ (e >> (25-11))
	mov	y2, f_		// y2 = f
	MY_ROR	y0, (11-6)	// y0 = (e >> (11-6)) ^ (e >> (25-6))

		vpsrld	XTMP5, XTMP2,   10	// XTMP5 = W[-2] >> 10 {DDCC}

	xor	y1, a_		// y1 = a ^ (a >> (22-13)
	xor	y2, g_		// y2 = f^g

		vpsrlq	XTMP3, XTMP2, 19	// XTMP3 = W[-2] MY_ROR 19 {xDxC}

	xor	y0, e_		// y0 = e ^ (e >> (11-6)) ^ (e >> (25-6))
	and	y2, e_		// y2 = (f^g)&e
	MY_ROR	y1, (13-2)	// y1 = (a >> (13-2)) ^ (a >> (22-2))

		vpsrlq	XTMP2, XTMP2, 17	// XTMP2 = W[-2] MY_ROR 17 {xDxC}

	xor	y1, a_		// y1 = a ^ (a >> (13-2)) ^ (a >> (22-2))
	MY_ROR	y0, 6		// y0 = S1 = (e>>6) & (e>>11) ^ (e>>25)
	xor	y2, g_		// y2 = CH = ((f^g)&e)^g

		vpxor	XTMP2, XTMP2, XTMP3

	MY_ROR	y1, 2		// y1 = S0 = (a>>2) ^ (a>>13) ^ (a>>22)
	add	y2, y0		// y2 = S1 + CH
	add	y2, [rsp + 3*4]	// y2 = k + w + S1 + CH
		vpxor	XTMP5, XTMP5, XTMP2	// XTMP5 = s1 {xDxC}
	mov	y0, a_		// y0 = a
	add	h_, y2		// h = h + S1 + CH + k + w
	mov	y2, a_		// y2 = a
		vpshufb	XTMP5, XTMP5, SHUF_DC00	// XTMP5 = s1 {DC00}
	or	y0, c_		// y0 = a|c
	add	d_, h_		// d = d + h + S1 + CH + k + w
	and	y2, c_		// y2 = a&c
		vpaddd	X0, XTMP5, XTMP0	// X0 = {W[3], W[2], W[1], W[0]}
	and	y0, b_		// y0 = (a|c)&b
	add	h_, y1		// h = h + S1 + CH + k + w + S0
	or	y0, y2		// y0 = MAJ = (a|c)&b)|(a&c)
	add	h_, y0		// h = h + S1 + CH + k + w + S0 + MAJ
ROTATE_ARGS
rotate_Xs
.endm

.macro DO_ROUND base offset
	mov	y0, e_		// y0 = e
	MY_ROR	y0, (25-11)	// y0 = e >> (25-11)
	mov	y1, a_		// y1 = a
	xor	y0, e_		// y0 = e ^ (e >> (25-11))
	MY_ROR	y1, (22-13)	// y1 = a >> (22-13)
	mov	y2, f_		// y2 = f
	xor	y1, a_		// y1 = a ^ (a >> (22-13)
	MY_ROR	y0, (11-6)	// y0 = (e >> (11-6)) ^ (e >> (25-6))
	xor	y2, g_		// y2 = f^g
	xor	y0, e_		// y0 = e ^ (e >> (11-6)) ^ (e >> (25-6))
	MY_ROR	y1, (13-2)	// y1 = (a >> (13-2)) ^ (a >> (22-2))
	and	y2, e_		// y2 = (f^g)&e
	xor	y1, a_		// y1 = a ^ (a >> (13-2)) ^ (a >> (22-2))
	MY_ROR	y0, 6		// y0 = S1 = (e>>6) & (e>>11) ^ (e>>25)
	xor	y2, g_		// y2 = CH = ((f^g)&e)^g
	add	y2, y0		// y2 = S1 + CH
	MY_ROR	y1, 2		// y1 = S0 = (a>>2) ^ (a>>13) ^ (a>>22)
	add	y2, [\base + \offset]	// y2 = k + w + S1 + CH
	mov	y0, a_		// y0 = a
	add	h_, y2		// h = h + S1 + CH + k + w
	mov	y2, a_		// y2 = a
	or	y0, c_		// y0 = a|c
	add	d_, h_		// d = d + h + S1 + CH + k + w
	and	y2, c_		// y2 = a&c
	and	y0, b_		// y0 = (a|c)&b
	add	h_, y1		// h = h + S1 + CH + k + w + S0
	or	y0, y2		// y0 = MAJ = (a|c)&b)|(a&c)
	add	h_, y0		// h = h + S1 + CH + k + w + S0 + MAJ
	ROTATE_ARGS
.endm

.global sha256_1_avx
#ifndef __WIN64__
.type   sha256_1_avx,%function
#endif
.align 32
sha256_1_avx:
        endbr64
	push	rbx
#ifdef __WIN64__
        push    r8
	push	rsi
	push	rdi
#endif
	push	rbp
        push    r12
	push	r13
	push	r14
	push	r15

	sub	rsp, stack_size
#ifdef __WIN64__
	vmovdqa	[rsp + _XMM_SAVE + 0*16],xmm6
	vmovdqa	[rsp + _XMM_SAVE + 1*16],xmm7
	vmovdqa	[rsp + _XMM_SAVE + 2*16],xmm8
	vmovdqa	[rsp + _XMM_SAVE + 3*16],xmm9
	vmovdqa	[rsp + _XMM_SAVE + 4*16],xmm10
	vmovdqa	[rsp + _XMM_SAVE + 5*16],xmm11
	vmovdqa	[rsp + _XMM_SAVE + 6*16],xmm12
	vmovdqa	[rsp + _XMM_SAVE + 7*16],xmm13
#endif
	vmovdqa	BYTE_FLIP_MASK, [rip + .LPSHUFFLE_BYTE_FLIP_MASK]
	vmovdqa	SHUF_00BA, [rip + .L_SHUF_00BA]
	vmovdqa	SHUF_DC00, [rip + .L_SHUF_DC00]

        shl         count, 5
        add         count, OUTPUT_PTR

.Lsha256_avx_1_block_loop:
        cmp     OUTPUT_PTR, count
        je      .Lsha256_1_avx_epilog

	//; load initial digest
	lea TBL,[rip + .LDIGEST]
	mov	a_, [TBL + 0*4]
	mov	b_, [TBL + 1*4]
	mov	c_, [TBL + 2*4]
	mov	d_, [TBL + 3*4]
	mov	e_, [TBL + 4*4]
	mov	f_, [TBL + 5*4]
	mov	g_, [TBL + 6*4]
	mov	h_, [TBL + 7*4]

	lea	TBL,[rip + .LK256]

	//; byte swap first 16 dwords
	COPY_XMM_AND_BSWAP	X0, [DATA_PTR + 0*16], BYTE_FLIP_MASK
	COPY_XMM_AND_BSWAP	X1, [DATA_PTR + 1*16], BYTE_FLIP_MASK
	COPY_XMM_AND_BSWAP	X2, [DATA_PTR + 2*16], BYTE_FLIP_MASK
	COPY_XMM_AND_BSWAP	X3, [DATA_PTR + 3*16], BYTE_FLIP_MASK

	//; schedule 48 input dwords, by doing 3 rounds of 16 each
.rept 3
.align 32
	vpaddd	XFER, X0, [TBL + 0*16]
	vmovdqa	[rsp], XFER
	FOUR_ROUNDS_AND_SCHED

	vpaddd	XFER, X0, [TBL + 1*16]
	vmovdqa	[rsp], XFER
	FOUR_ROUNDS_AND_SCHED

	vpaddd	XFER, X0, [TBL + 2*16]
	vmovdqa	[rsp], XFER
	FOUR_ROUNDS_AND_SCHED

	vpaddd	XFER, X0, [TBL + 3*16]
	vmovdqa	[rsp], XFER
	add	TBL, 4*16
	FOUR_ROUNDS_AND_SCHED
.endr

.rept 2
	vpaddd	XFER, X0, [TBL + 0*16]
	vmovdqa	[rsp], XFER
	DO_ROUND rsp, 0
	DO_ROUND rsp, 4
	DO_ROUND rsp, 8
	DO_ROUND rsp, 12

	vpaddd	XFER, X1, [TBL + 1*16]
	vmovdqa	[rsp], XFER
	add	TBL, 2*16
	DO_ROUND rsp, 0
	DO_ROUND rsp, 4
	DO_ROUND rsp, 8
	DO_ROUND rsp, 12

	vmovdqa	X0, X2
	vmovdqa	X1, X3
.endr

        // add old digest

	lea TBL,[rip + .LDIGEST]
	add	a_, [TBL + 0*4]
	add	b_, [TBL + 1*4]
	add	c_, [TBL + 2*4]
	add	d_, [TBL + 3*4]
	add	e_, [TBL + 4*4]
	add	f_, [TBL + 5*4]
	add	g_, [TBL + 6*4]
	add	h_, [TBL + 7*4]


        // rounds with padding

        // save old digest
        //
        mov    [rsp + _DIGEST + 0*4], a_
        mov    [rsp + _DIGEST + 1*4], b_
        mov    [rsp + _DIGEST + 2*4], c_
        mov    [rsp + _DIGEST + 3*4], d_
        mov    [rsp + _DIGEST + 4*4], e_
        mov    [rsp + _DIGEST + 5*4], f_
        mov    [rsp + _DIGEST + 6*4], g_
        mov    [rsp + _DIGEST + 7*4], h_

        lea     TBL,[rip + .LPADDING]

.set .Li, 0
.rept 64
                    DO_ROUND TBL, .Li
	            .set .Li, .Li+4
.endr

        //; add the previous digest
        add   a_, [rsp + _DIGEST + 0*4]
        add   b_, [rsp + _DIGEST + 1*4]
        add   c_, [rsp + _DIGEST + 2*4]
        add   d_, [rsp + _DIGEST + 3*4]
        add   e_, [rsp + _DIGEST + 4*4]
        add   f_, [rsp + _DIGEST + 5*4]
        add   g_, [rsp + _DIGEST + 6*4]
        add   h_, [rsp + _DIGEST + 7*4]

        //; shuffle the bytes to little endian
        bswap  a_
        bswap  b_
        bswap  c_
        bswap  d_
        bswap  e_
        bswap  f_
        bswap  g_
        bswap  h_

        //; write resulting hash
        mov   [OUTPUT_PTR + 0*4], a_
        mov   [OUTPUT_PTR + 1*4], b_
        mov   [OUTPUT_PTR + 2*4], c_
        mov   [OUTPUT_PTR + 3*4], d_
        mov   [OUTPUT_PTR + 4*4], e_
        mov   [OUTPUT_PTR + 5*4], f_
        mov   [OUTPUT_PTR + 6*4], g_
        mov   [OUTPUT_PTR + 7*4], h_

        add   OUTPUT_PTR, 32
        add   DATA_PTR, 64
        jmp   .Lsha256_avx_1_block_loop

.Lsha256_1_avx_epilog:

#ifdef __WIN64__
	vmovdqa	xmm6,[rsp + _XMM_SAVE + 0*16]
	vmovdqa	xmm7,[rsp + _XMM_SAVE + 1*16]
	vmovdqa	xmm8,[rsp + _XMM_SAVE + 2*16]
	vmovdqa	xmm9,[rsp + _XMM_SAVE + 3*16]
	vmovdqa	xmm10,[rsp + _XMM_SAVE + 4*16]
	vmovdqa	xmm11,[rsp + _XMM_SAVE + 5*16]
	vmovdqa	xmm12,[rsp + _XMM_SAVE + 6*16]
	vmovdqa	xmm13,[rsp + _XMM_SAVE + 7*16]
#endif

	add	rsp, stack_size

	pop	r15
	pop	r14
	pop	r13
        pop     r12
	pop	rbp
#ifdef __WIN64__
	pop	rdi
	pop	rsi
        pop     r8
#endif
	pop	rbx

	ret
#ifdef __linux__
.size sha256_1_avx,.-sha256_1_avx
.section .note.GNU-stack,"",@progbits
#endif
#endif
