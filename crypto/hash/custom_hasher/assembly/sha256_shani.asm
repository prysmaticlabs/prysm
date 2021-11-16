;;  sha256.asm
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
;; Copyright (c) 2013, Intel Corporation 
; 
; All rights reserved. 
; 
; Redistribution and use in source and binary forms, with or without
; modification, are permitted provided that the following conditions are
; met: 
; 
; * Redistributions of source code must retain the above copyright
;   notice, this list of conditions and the following disclaimer.  
; 
; * Redistributions in binary form must reproduce the above copyright
;   notice, this list of conditions and the following disclaimer in the
;   documentation and/or other materials provided with the
;   distribution. 
; 
; * Neither the name of the Intel Corporation nor the names of its
;   contributors may be used to endorse or promote products derived from
;   this software without specific prior written permission. 
; 
; 
; THIS SOFTWARE IS PROVIDED BY INTEL CORPORATION ""AS IS"" AND ANY
; EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
; IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
; PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL INTEL CORPORATION OR
; CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
; EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
; PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
; PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
; LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
; NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
; SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
%ifdef WINABI
%define OUTPUT_PTR	rcx 	; 1st arg
%define DATA_PTR	rdx 	; 2nd arg
%define NUM_BLKS 	r8	; 3rd arg
%define SHA256PADDING   rdi
%else
%define OUTPUT_PTR	rdi	; 1st arg
%define DATA_PTR	rsi	; 2nd arg
%define NUM_BLKS	rdx	; 3rd arg
%define SHA256PADDING   rcx
%endif

%define SHA256CONSTANTS	rax


%ifdef WINABI
%define RSPSAVE		r9

struc frame
.xmm_save:	resdq	5
endstruc
%endif

%define MSG		xmm0
%define STATE0		xmm1
%define STATE1		xmm2
%define MSGTMP0		xmm3
%define MSGTMP1		xmm4
%define MSGTMP2		xmm5
%define MSGTMP3		xmm6
%define MSGTMP4		xmm7

%define SHUF_MASK	xmm8

%define ABEF_SAVE	xmm9
%define CDGH_SAVE	xmm10

%define STATE0b         xmm9
%define STATE1b         xmm10
%define MSGTMP0b	xmm11
%define MSGTMP1b	xmm12
%define MSGTMP2b	xmm13
%define MSGTMP3b	xmm14
%define MSGTMP4b	xmm15




;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
;; void sha256_update(uint32_t *output, const void *data, uint32_t numBlocks);
;; arg 1 : pointer to output digest
;; arg 2 : pointer to input data
;; arg 3 : Number of blocks to process
;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
section .text
global sha256_shani:function
align 32
sha256_shani:
%ifdef WINABI
	mov		RSPSAVE, rsp
	sub		rsp, frame_size
	and		rsp, ~0xF

	movdqa		[rsp + frame.xmm_save + 0*16], xmm6
	movdqa		[rsp + frame.xmm_save + 1*16], xmm7
	movdqa		[rsp + frame.xmm_save + 2*16], xmm8
	movdqa		[rsp + frame.xmm_save + 3*16], xmm9
	movdqa		[rsp + frame.xmm_save + 4*16], xmm10
%endif

	movdqa		SHUF_MASK, [PSHUFFLE_BYTE_FLIP_MASK wrt rip]
	lea		SHA256CONSTANTS,[K256 wrt rip]
	lea		SHA256PADDING,[PADDING wrt rip]


.hash_2_blocks:
        cmp             NUM_BLKS, 2
	jl		.hash_1_block

        movdqa          STATE0, [DIGEST wrt rip]
        movdqa          STATE1, [DIGEST + 16 wrt rip]

        movdqa          STATE0b, [DIGEST wrt rip]
        movdqa          STATE1b, [DIGEST + 16 wrt rip]

	;; Rounds 0-3
	movdqu		MSG, [DATA_PTR + 0*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP0, MSG
		paddd		MSG, [SHA256CONSTANTS + 0*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	movdqu		MSG, [DATA_PTR + 4*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP0b, MSG
		paddd		MSG, [SHA256CONSTANTS + 0*16]
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b


	;; Rounds 4-7
	movdqu		MSG, [DATA_PTR + 1*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP1, MSG
		paddd		MSG, [SHA256CONSTANTS + 1*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP0, MSGTMP1
	movdqu		MSG, [DATA_PTR + 5*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP1b, MSG
		paddd		MSG, [SHA256CONSTANTS + 1*16]
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP0b, MSGTMP1b
	;; Rounds 8-11
	movdqu		MSG, [DATA_PTR + 2*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP2, MSG
		paddd		MSG, [SHA256CONSTANTS + 2*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP1, MSGTMP2
	movdqu		MSG, [DATA_PTR + 6*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP2b, MSG
		paddd		MSG, [SHA256CONSTANTS + 2*16]
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP1b, MSGTMP2b
	;; Rounds 12-15
	movdqu		MSG, [DATA_PTR + 3*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP3, MSG
		paddd		MSG, [SHA256CONSTANTS + 3*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP3
	palignr		MSGTMP4, MSGTMP2, 4
	paddd		MSGTMP0, MSGTMP4
	sha256msg2	MSGTMP0, MSGTMP3
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP2, MSGTMP3
	movdqu		MSG, [DATA_PTR + 7*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP3b, MSG
		paddd		MSG, [SHA256CONSTANTS + 3*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP3b
	palignr		MSGTMP4b, MSGTMP2b, 4
	paddd		MSGTMP0b, MSGTMP4b
	sha256msg2	MSGTMP0b, MSGTMP3b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP2b, MSGTMP3b

	;; Rounds 16-19
	movdqa		MSG, MSGTMP0
		paddd		MSG, [SHA256CONSTANTS + 4*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP0
	palignr		MSGTMP4, MSGTMP3, 4
	paddd		MSGTMP1, MSGTMP4
	sha256msg2	MSGTMP1, MSGTMP0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP3, MSGTMP0
	movdqa		MSG, MSGTMP0b
		paddd		MSG, [SHA256CONSTANTS + 4*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP0b
	palignr		MSGTMP4b, MSGTMP3b, 4
	paddd		MSGTMP1b, MSGTMP4b
	sha256msg2	MSGTMP1b, MSGTMP0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP3b, MSGTMP0b

	;; Rounds 20-23
	movdqa		MSG, MSGTMP1
		paddd		MSG, [SHA256CONSTANTS + 5*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP1
	palignr		MSGTMP4, MSGTMP0, 4
	paddd		MSGTMP2, MSGTMP4
	sha256msg2	MSGTMP2, MSGTMP1
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP0, MSGTMP1
	movdqa		MSG, MSGTMP1b
		paddd		MSG, [SHA256CONSTANTS + 5*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP1b
	palignr		MSGTMP4b, MSGTMP0b, 4
	paddd		MSGTMP2b, MSGTMP4b
	sha256msg2	MSGTMP2b, MSGTMP1b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP0b, MSGTMP1b

	;; Rounds 24-27
	movdqa		MSG, MSGTMP2
		paddd		MSG, [SHA256CONSTANTS + 6*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP2
	palignr		MSGTMP4, MSGTMP1, 4
	paddd		MSGTMP3, MSGTMP4
	sha256msg2	MSGTMP3, MSGTMP2
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP1, MSGTMP2

	movdqa		MSG, MSGTMP2b
		paddd		MSG, [SHA256CONSTANTS + 6*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP2b
	palignr		MSGTMP4b, MSGTMP1b, 4
	paddd		MSGTMP3b, MSGTMP4b
	sha256msg2	MSGTMP3b, MSGTMP2b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP1b, MSGTMP2b

	;; Rounds 28-31
	movdqa		MSG, MSGTMP3
		paddd		MSG, [SHA256CONSTANTS + 7*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP3
	palignr		MSGTMP4, MSGTMP2, 4
	paddd		MSGTMP0, MSGTMP4
	sha256msg2	MSGTMP0, MSGTMP3
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP2, MSGTMP3
	movdqa		MSG, MSGTMP3b
		paddd		MSG, [SHA256CONSTANTS + 7*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP3b
	palignr		MSGTMP4b, MSGTMP2b, 4
	paddd		MSGTMP0b, MSGTMP4b
	sha256msg2	MSGTMP0b, MSGTMP3b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP2b, MSGTMP3b
	;; Rounds 32-35
	movdqa		MSG, MSGTMP0
		paddd		MSG, [SHA256CONSTANTS + 8*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP0
	palignr		MSGTMP4, MSGTMP3, 4
	paddd		MSGTMP1, MSGTMP4
	sha256msg2	MSGTMP1, MSGTMP0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP3, MSGTMP0
	movdqa		MSG, MSGTMP0b
		paddd		MSG, [SHA256CONSTANTS + 8*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP0b
	palignr		MSGTMP4b, MSGTMP3b, 4
	paddd		MSGTMP1b, MSGTMP4b
	sha256msg2	MSGTMP1b, MSGTMP0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP3b, MSGTMP0b

	;; Rounds 36-39
	movdqa		MSG, MSGTMP1
		paddd		MSG, [SHA256CONSTANTS + 9*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP1
	palignr		MSGTMP4, MSGTMP0, 4
	paddd		MSGTMP2, MSGTMP4
	sha256msg2	MSGTMP2, MSGTMP1
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP0, MSGTMP1
	movdqa		MSG, MSGTMP1b
		paddd		MSG, [SHA256CONSTANTS + 9*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP1b
	palignr		MSGTMP4b, MSGTMP0b, 4
	paddd		MSGTMP2b, MSGTMP4b
	sha256msg2	MSGTMP2b, MSGTMP1b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP0b, MSGTMP1b

	;; Rounds 40-43
	movdqa		MSG, MSGTMP2
		paddd		MSG, [SHA256CONSTANTS + 10*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP2
	palignr		MSGTMP4, MSGTMP1, 4
	paddd		MSGTMP3, MSGTMP4
	sha256msg2	MSGTMP3, MSGTMP2
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP1, MSGTMP2
	movdqa		MSG, MSGTMP2b
		paddd		MSG, [SHA256CONSTANTS + 10*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP2b
	palignr		MSGTMP4b, MSGTMP1b, 4
	paddd		MSGTMP3b, MSGTMP4b
	sha256msg2	MSGTMP3b, MSGTMP2b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP1b, MSGTMP2b

	;; Rounds 44-47
	movdqa		MSG, MSGTMP3
		paddd		MSG, [SHA256CONSTANTS + 11*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP3
	palignr		MSGTMP4, MSGTMP2, 4
	paddd		MSGTMP0, MSGTMP4
	sha256msg2	MSGTMP0, MSGTMP3
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP2, MSGTMP3
	movdqa		MSG, MSGTMP3b
		paddd		MSG, [SHA256CONSTANTS + 11*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP3b
	palignr		MSGTMP4b, MSGTMP2b, 4
	paddd		MSGTMP0b, MSGTMP4b
	sha256msg2	MSGTMP0b, MSGTMP3b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP2b, MSGTMP3b
       
	;; Rounds 48-51
	movdqa		MSG, MSGTMP0
		paddd		MSG, [SHA256CONSTANTS + 12*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP0
	palignr		MSGTMP4, MSGTMP3, 4
	paddd		MSGTMP1, MSGTMP4
	sha256msg2	MSGTMP1, MSGTMP0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP3, MSGTMP0
	movdqa		MSG, MSGTMP0b
		paddd		MSG, [SHA256CONSTANTS + 12*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP0b
	palignr		MSGTMP4b, MSGTMP3b, 4
	paddd		MSGTMP1b, MSGTMP4b
	sha256msg2	MSGTMP1b, MSGTMP0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b
	sha256msg1	MSGTMP3b, MSGTMP0b
        
 	;; Rounds 52-55
	movdqa		MSG, MSGTMP1
		paddd		MSG, [SHA256CONSTANTS + 13*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP1
	palignr		MSGTMP4, MSGTMP0, 4
	paddd		MSGTMP2, MSGTMP4
	sha256msg2	MSGTMP2, MSGTMP1
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	movdqa		MSG, MSGTMP1b
		paddd		MSG, [SHA256CONSTANTS + 13*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP1b
	palignr		MSGTMP4b, MSGTMP0b, 4
	paddd		MSGTMP2b, MSGTMP4b
	sha256msg2	MSGTMP2b, MSGTMP1b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b

	;; Rounds 56-59
	movdqa		MSG, MSGTMP2
		paddd		MSG, [SHA256CONSTANTS + 14*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP2
	palignr		MSGTMP4, MSGTMP1, 4
	paddd		MSGTMP3, MSGTMP4
	sha256msg2	MSGTMP3, MSGTMP2
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	movdqa		MSG, MSGTMP2b
		paddd		MSG, [SHA256CONSTANTS + 14*16]
		sha256rnds2	STATE1b, STATE0b
	movdqa		MSGTMP4b, MSGTMP2b
	palignr		MSGTMP4b, MSGTMP1b, 4
	paddd		MSGTMP3b, MSGTMP4b
	sha256msg2	MSGTMP3b, MSGTMP2b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b


	;; Rounds 60-63
	movdqa		MSG, MSGTMP3
		paddd		MSG, [SHA256CONSTANTS + 15*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	movdqa		MSG, MSGTMP3b
		paddd		MSG, [SHA256CONSTANTS + 15*16]
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0b, STATE1b

       
	paddd		STATE0, [DIGEST wrt rip]
	paddd		STATE1, [DIGEST + 16 wrt rip]
	paddd           STATE0b, [DIGEST wrt rip]
	paddd		STATE1b, [DIGEST + 16 wrt rip]

        ;; Rounds with PADDING
        
	;; Save hash values for addition after rounds
	movdqa		[ABEF_SAVEa wrt rip], STATE0
	movdqa		[CDGH_SAVEa wrt rip], STATE1
	movdqa		[ABEF_SAVEb wrt rip], STATE0b
	movdqa		[CDGH_SAVEb wrt rip], STATE1b

	;; Rounds 0-3
	movdqa		MSG, [SHA256PADDING + 0*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b

	;; Rounds 4-7
	movdqa		MSG, [SHA256PADDING + 1*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 2*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 3*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 4*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 5*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 6*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 7*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 8*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 9*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b

	movdqa		MSG, [SHA256PADDING + 10*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 11*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 12*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 13*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b

	movdqa		MSG, [SHA256PADDING + 14*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b


	movdqa		MSG, [SHA256PADDING + 15*16]
		sha256rnds2	STATE1, STATE0
		sha256rnds2	STATE1b, STATE0b
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
		sha256rnds2	STATE0b, STATE1b

	paddd		STATE0, [ABEF_SAVEa wrt rip]
	paddd		STATE1, [CDGH_SAVEa wrt rip]
	paddd           STATE0b, [ABEF_SAVEb wrt rip]
	paddd		STATE1b, [CDGH_SAVEb wrt rip]


	;; Write hash values back in the correct order
	pshufd		STATE0,  STATE0,  0x1B	; FEBA
	pshufd		STATE1,  STATE1,  0xB1	; DCHG
	pshufd		STATE0b,  STATE0b,  0x1B	; FEBA
	pshufd		STATE1b,  STATE1b,  0xB1	; DCHG
	movdqa		MSGTMP4, STATE0
	movdqa		MSGTMP4b, STATE0b
	pblendw		STATE0,  STATE1,  0xF0	; DCBA
	pblendw		STATE0b,  STATE1b,  0xF0	; DCBA
	palignr		STATE1,  MSGTMP4, 8	; HGFE
	palignr		STATE1b,  MSGTMP4b, 8	; HGFE

        pshufb          STATE0, SHUF_MASK
        pshufb          STATE0b, SHUF_MASK
        pshufb          STATE1, SHUF_MASK
        pshufb          STATE1b, SHUF_MASK


	movdqu		[OUTPUT_PTR + 0*16], STATE0
	movdqu		[OUTPUT_PTR + 1*16], STATE1
	movdqu		[OUTPUT_PTR + 2*16], STATE0b
	movdqu		[OUTPUT_PTR + 3*16], STATE1b

	;; Increment data pointer and loop if more to process
	add		DATA_PTR, 128
        add             OUTPUT_PTR, 64
        
	sub		NUM_BLKS,2
        jmp             .hash_2_blocks

.hash_1_block:

        test            NUM_BLKS,NUM_BLKS
        jz              .done_hash

        movdqa          STATE0, [DIGEST wrt rip]
        movdqa          STATE1, [DIGEST + 16 wrt rip]

	;; Save hash values for addition after rounds
	movdqa		ABEF_SAVE, STATE0
	movdqa		CDGH_SAVE, STATE1

	;; Rounds 0-3
	movdqu		MSG, [DATA_PTR + 0*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP0, MSG
		paddd		MSG, [SHA256CONSTANTS + 0*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 4-7
	movdqu		MSG, [DATA_PTR + 1*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP1, MSG
		paddd		MSG, [SHA256CONSTANTS + 1*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP0, MSGTMP1

	;; Rounds 8-11
	movdqu		MSG, [DATA_PTR + 2*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP2, MSG
		paddd		MSG, [SHA256CONSTANTS + 2*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP1, MSGTMP2

	;; Rounds 12-15
	movdqu		MSG, [DATA_PTR + 3*16]
	pshufb		MSG, SHUF_MASK
	movdqa		MSGTMP3, MSG
		paddd		MSG, [SHA256CONSTANTS + 3*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP3
	palignr		MSGTMP4, MSGTMP2, 4
	paddd		MSGTMP0, MSGTMP4
	sha256msg2	MSGTMP0, MSGTMP3
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP2, MSGTMP3

	;; Rounds 16-19
	movdqa		MSG, MSGTMP0
		paddd		MSG, [SHA256CONSTANTS + 4*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP0
	palignr		MSGTMP4, MSGTMP3, 4
	paddd		MSGTMP1, MSGTMP4
	sha256msg2	MSGTMP1, MSGTMP0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP3, MSGTMP0

	;; Rounds 20-23
	movdqa		MSG, MSGTMP1
		paddd		MSG, [SHA256CONSTANTS + 5*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP1
	palignr		MSGTMP4, MSGTMP0, 4
	paddd		MSGTMP2, MSGTMP4
	sha256msg2	MSGTMP2, MSGTMP1
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP0, MSGTMP1

	;; Rounds 24-27
	movdqa		MSG, MSGTMP2
		paddd		MSG, [SHA256CONSTANTS + 6*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP2
	palignr		MSGTMP4, MSGTMP1, 4
	paddd		MSGTMP3, MSGTMP4
	sha256msg2	MSGTMP3, MSGTMP2
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP1, MSGTMP2

	;; Rounds 28-31
	movdqa		MSG, MSGTMP3
		paddd		MSG, [SHA256CONSTANTS + 7*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP3
	palignr		MSGTMP4, MSGTMP2, 4
	paddd		MSGTMP0, MSGTMP4
	sha256msg2	MSGTMP0, MSGTMP3
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP2, MSGTMP3

	;; Rounds 32-35
	movdqa		MSG, MSGTMP0
		paddd		MSG, [SHA256CONSTANTS + 8*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP0
	palignr		MSGTMP4, MSGTMP3, 4
	paddd		MSGTMP1, MSGTMP4
	sha256msg2	MSGTMP1, MSGTMP0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP3, MSGTMP0

	;; Rounds 36-39
	movdqa		MSG, MSGTMP1
		paddd		MSG, [SHA256CONSTANTS + 9*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP1
	palignr		MSGTMP4, MSGTMP0, 4
	paddd		MSGTMP2, MSGTMP4
	sha256msg2	MSGTMP2, MSGTMP1
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP0, MSGTMP1

	;; Rounds 40-43
	movdqa		MSG, MSGTMP2
		paddd		MSG, [SHA256CONSTANTS + 10*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP2
	palignr		MSGTMP4, MSGTMP1, 4
	paddd		MSGTMP3, MSGTMP4
	sha256msg2	MSGTMP3, MSGTMP2
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP1, MSGTMP2

	;; Rounds 44-47
	movdqa		MSG, MSGTMP3
		paddd		MSG, [SHA256CONSTANTS + 11*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP3
	palignr		MSGTMP4, MSGTMP2, 4
	paddd		MSGTMP0, MSGTMP4
	sha256msg2	MSGTMP0, MSGTMP3
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP2, MSGTMP3

	;; Rounds 48-51
	movdqa		MSG, MSGTMP0
		paddd		MSG, [SHA256CONSTANTS + 12*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP0
	palignr		MSGTMP4, MSGTMP3, 4
	paddd		MSGTMP1, MSGTMP4
	sha256msg2	MSGTMP1, MSGTMP0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1
	sha256msg1	MSGTMP3, MSGTMP0

	;; Rounds 52-55
	movdqa		MSG, MSGTMP1
		paddd		MSG, [SHA256CONSTANTS + 13*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP1
	palignr		MSGTMP4, MSGTMP0, 4
	paddd		MSGTMP2, MSGTMP4
	sha256msg2	MSGTMP2, MSGTMP1
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 56-59
	movdqa		MSG, MSGTMP2
		paddd		MSG, [SHA256CONSTANTS + 14*16]
		sha256rnds2	STATE1, STATE0
	movdqa		MSGTMP4, MSGTMP2
	palignr		MSGTMP4, MSGTMP1, 4
	paddd		MSGTMP3, MSGTMP4
	sha256msg2	MSGTMP3, MSGTMP2
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 60-63
	movdqa		MSG, MSGTMP3
		paddd		MSG, [SHA256CONSTANTS + 15*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Add current hash values with previously saved
	paddd		STATE0, ABEF_SAVE
	paddd		STATE1, CDGH_SAVE

        ;; Rounds with PADDING
	;; Save hash values for addition after rounds
	movdqa		ABEF_SAVE, STATE0
	movdqa		CDGH_SAVE, STATE1

	;; Rounds 0-3
	movdqa		MSG, [SHA256PADDING + 0*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 4-7
	movdqa		MSG, [SHA256PADDING + 1*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 8-11
	movdqa		MSG, [SHA256PADDING + 2*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 12-15
	movdqa		MSG, [SHA256PADDING + 3*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 16-19
	movdqa		MSG, [SHA256PADDING + 4*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 20-23
	movdqa		MSG, [SHA256PADDING + 5*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 24-27
	movdqa		MSG, [SHA256PADDING + 6*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 28-31
	movdqa		MSG, [SHA256PADDING + 7*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 32-35
	movdqa		MSG, [SHA256PADDING + 8*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 36-39
	movdqa		MSG, [SHA256PADDING + 9*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 40-43
	movdqa		MSG, [SHA256PADDING + 10*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 44-47
	movdqa		MSG, [SHA256PADDING + 11*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 48-51
	movdqa		MSG, [SHA256PADDING + 12*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 52-55
	movdqa		MSG, [SHA256PADDING + 13*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 56-59
	movdqa		MSG, [SHA256PADDING + 14*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Rounds 60-63
	movdqa		MSG, [SHA256PADDING + 15*16]
		sha256rnds2	STATE1, STATE0
		pshufd 		MSG, MSG, 0x0E
		sha256rnds2	STATE0, STATE1

	;; Add current hash values with previously saved
	paddd		STATE0, ABEF_SAVE
	paddd		STATE1, CDGH_SAVE

	;; Write hash values back in the correct order
	pshufd		STATE0,  STATE0,  0x1B	; FEBA
	pshufd		STATE1,  STATE1,  0xB1	; DCHG
	movdqa		MSGTMP4, STATE0
	pblendw		STATE0,  STATE1,  0xF0	; DCBA
	palignr		STATE1,  MSGTMP4, 8	; HGFE

        pshufb          STATE0, SHUF_MASK
        pshufb          STATE1, SHUF_MASK

	movdqu		[OUTPUT_PTR + 0*16], STATE0
	movdqu		[OUTPUT_PTR + 1*16], STATE1

.done_hash:
%ifdef WINABI
	movdqa		xmm6,  [rsp + frame.xmm_save + 0*16]
	movdqa		xmm7,  [rsp + frame.xmm_save + 1*16]
	movdqa		xmm8,  [rsp + frame.xmm_save + 2*16]
	movdqa		xmm9,  [rsp + frame.xmm_save + 3*16]
	movdqa		xmm10, [rsp + frame.xmm_save + 4*16]
	mov		rsp, RSPSAVE
%endif

	ret	
	
section .rodata
align 64
K256:
	dd	0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5
	dd	0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5
	dd	0xd807aa98,0x12835b01,0x243185be,0x550c7dc3
	dd	0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174
	dd	0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc
	dd	0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da
	dd	0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7
	dd	0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967
	dd	0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13
	dd	0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85
	dd	0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3
	dd	0xd192e819,0xd6990624,0xf40e3585,0x106aa070
	dd	0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5
	dd	0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3
	dd	0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208
	dd	0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2

PSHUFFLE_BYTE_FLIP_MASK: ddq 0x0c0d0e0f08090a0b0405060700010203

DIGEST:
        ddq      0x6a09e667bb67ae85510e527f9b05688c
        ddq      0x3c6ef372a54ff53a1f83d9ab5be0cd19

PADDING:
        dd      0xc28a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5
        dd      0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5
        dd      0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3
        dd      0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf374
        dd      0x649b69c1, 0xf0fe4786, 0xfe1edc6, 0x240cf254
        dd      0x4fe9346f, 0x6cc984be, 0x61b9411e, 0x16f988fa
        dd      0xf2c65152, 0xa88e5a6d, 0xb019fc65, 0xb9d99ec7
        dd      0x9a1231c3, 0xe70eeaa0, 0xfdb1232b, 0xc7353eb0
        dd      0x3069bad5, 0xcb976d5f, 0x5a0f118f, 0xdc1eeefd
        dd      0xa35b689, 0xde0b7a04, 0x58f4ca9d, 0xe15d5b16
        dd      0x7f3e86, 0x37088980, 0xa507ea32, 0x6fab9537
        dd      0x17406110, 0xd8cd6f1, 0xcdaa3b6d, 0xc0bbbe37
        dd      0x83613bda, 0xdb48a363, 0xb02e931, 0x6fd15ca7
        dd      0x521afaca, 0x31338431, 0x6ed41a95, 0x6d437890
        dd      0xc39c91f2, 0x9eccabbd, 0xb5c9a0e6, 0x532fb63c
        dd      0xd2c741c6, 0x7237ea3, 0xa4954b68, 0x4c191d76

section .data

align 64

ABEF_SAVEa:     dd      0x9b05688c, 0x510e527f, 0xbb67ae85, 0x6a09e667
CDGH_SAVEa:     dd      0x5be0cd19, 0x1f83d9ab, 0xa54ff53a, 0x3c6ef372
ABEF_SAVEb:     dd      0x9b05688c, 0x510e527f, 0xbb67ae85, 0x6a09e667
CDGH_SAVEb:     dd      0x5be0cd19, 0x1f83d9ab, 0xa54ff53a, 0x3c6ef372

