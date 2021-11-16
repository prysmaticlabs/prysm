#ifndef __CUSTOM_HASHER__
#define __CUSTOM_HASHER__

#include <stdint.h>
extern void sha256_4_avx(unsigned char* output, const unsigned char* input, uint64_t blocks);
extern void sha256_8_avx2(unsigned char* output, const unsigned char* input, uint64_t blocks);
extern void sha256_shani(unsigned char* output, const unsigned char* input, uint64_t blocks);
#endif
