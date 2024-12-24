/* src/libsecp256k1-config.h.  Generated from libsecp256k1-config.h.in by configure.  */
/* src/libsecp256k1-config.h.in.  Generated from configure.ac by autoheader.  */

#ifndef LIBSECP256K1_CONFIG_H

#define LIBSECP256K1_CONFIG_H

/* Define this symbol to compile out all VERIFY code */
/* #undef COVERAGE */

/* Set ecmult gen precision bits */
#define ECMULT_GEN_PREC_BITS 4

/* Set window size for ecmult precomputation */
#define ECMULT_WINDOW_SIZE 15

/* Define this symbol to enable the ECDH module */
/* #undef ENABLE_MODULE_ECDH */

/* Define this symbol to enable the extrakeys module */
#define ENABLE_MODULE_EXTRAKEYS 1

/* Define this symbol to enable the ECDSA pubkey recovery module */
#define ENABLE_MODULE_RECOVERY 1

/* Define this symbol to enable the schnorrsig module */
#define ENABLE_MODULE_SCHNORRSIG 1

/* Define to 1 if you have the <dlfcn.h> header file. */
#define HAVE_DLFCN_H 1

/* Define to 1 if you have the <inttypes.h> header file. */
#define HAVE_INTTYPES_H 1

/* Define to 1 if you have the <stdint.h> header file. */
#define HAVE_STDINT_H 1

/* Define to 1 if you have the <stdio.h> header file. */
#define HAVE_STDIO_H 1

/* Define to 1 if you have the <stdlib.h> header file. */
#define HAVE_STDLIB_H 1

/* Define to 1 if you have the <strings.h> header file. */
#define HAVE_STRINGS_H 1

/* Define to 1 if you have the <string.h> header file. */
#define HAVE_STRING_H 1

/* Define to 1 if you have the <sys/stat.h> header file. */
#define HAVE_SYS_STAT_H 1

/* Define to 1 if you have the <sys/types.h> header file. */
#define HAVE_SYS_TYPES_H 1

/* Define to 1 if you have the <unistd.h> header file. */
#define HAVE_UNISTD_H 1

/* Define this symbol if valgrind is installed, and it supports the host
   platform */
/* #undef HAVE_VALGRIND */

/* Define to the sub-directory where libtool stores uninstalled libraries. */
#define LT_OBJDIR ".libs/"

/* Name of package */
#define PACKAGE "libsecp256k1"

/* Define to the address where bug reports for this package should be sent. */
#define PACKAGE_BUGREPORT "https://github.com/bitcoin-core/secp256k1/issues"

/* Define to the full name of this package. */
#define PACKAGE_NAME "libsecp256k1"

/* Define to the full name and version of this package. */
#define PACKAGE_STRING "libsecp256k1 0.1.0-pre"

/* Define to the one symbol short name of this package. */
#define PACKAGE_TARNAME "libsecp256k1"

/* Define to the home page for this package. */
#define PACKAGE_URL "https://github.com/bitcoin-core/secp256k1"

/* Define to the version of this package. */
#define PACKAGE_VERSION "0.1.0-pre"

/* Define to 1 if all of the C90 standard headers exist (not just the ones
   required in a freestanding environment). This macro is provided for
   backward compatibility; new code need not use it. */
#define STDC_HEADERS 1

/* Define this symbol to enable x86_64 assembly optimizations */
#define USE_ASM_X86_64 1

/* Define this symbol if an external (non-inline) assembly implementation is
   used */
/* #undef USE_EXTERNAL_ASM */

/* Define this symbol if an external implementation of the default callbacks
   is used */
/* #undef USE_EXTERNAL_DEFAULT_CALLBACKS */

/* Define this symbol to force the use of the (unsigned) __int128 based wide
   multiplication implementation */
/* #undef USE_FORCE_WIDEMUL_INT128 */

/* Define this symbol to force the use of the (u)int64_t based wide
   multiplication implementation */
/* #undef USE_FORCE_WIDEMUL_INT64 */

/* Version number of package */
#define VERSION "0.1.0-pre"

#endif /*LIBSECP256K1_CONFIG_H*/
