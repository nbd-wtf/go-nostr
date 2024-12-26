list:
    @just --list

vendor-libsecp256k1:
    #!/usr/bin/env fish
    rm -r libsecp256k1
    mkdir libsecp256k1
    mkdir libsecp256k1/include
    mkdir libsecp256k1/src
    mkdir libsecp256k1/src/asm
    mkdir libsecp256k1/src/modules
    mkdir libsecp256k1/src/modules/extrakeys
    mkdir libsecp256k1/src/modules/schnorrsig

    wget https://api.github.com/repos/bitcoin-core/secp256k1/tarball/v0.6.0 -O libsecp256k1.tar.gz
    tar -xvf libsecp256k1.tar.gz
    rm libsecp256k1.tar.gz
    cd bitcoin-core-secp256k1-*
    for f in include/secp256k1.h include/secp256k1_ecdh.h include/secp256k1_ellswift.h include/secp256k1_extrakeys.h include/secp256k1_preallocated.h include/secp256k1_recovery.h include/secp256k1_schnorrsig.h src/asm/field_10x26_arm.s src/assumptions.h src/bench.c src/bench.h src/bench_ecmult.c src/bench_internal.c src/checkmem.h src/ecdsa.h src/ecdsa_impl.h src/eckey.h src/eckey_impl.h src/ecmult.h src/ecmult_compute_table.h src/ecmult_compute_table_impl.h src/ecmult_const.h src/ecmult_const_impl.h src/ecmult_gen.h src/ecmult_gen_compute_table.h src/ecmult_gen_compute_table_impl.h src/ecmult_gen_impl.h src/ecmult_impl.h src/field.h src/field_10x26.h src/field_10x26_impl.h src/field_5x52.h src/field_5x52_impl.h src/field_5x52_int128_impl.h src/field_impl.h src/group.h src/group_impl.h src/hash.h src/hash_impl.h src/hsort.h src/hsort_impl.h src/int128.h src/int128_impl.h src/int128_native.h src/int128_native_impl.h src/int128_struct.h src/int128_struct_impl.h src/modinv32.h src/modinv32_impl.h src/modinv64.h src/modinv64_impl.h src/modules/extrakeys/main_impl.h src/modules/schnorrsig/main_impl.h src/precompute_ecmult.c src/precompute_ecmult_gen.c src/precomputed_ecmult.c src/precomputed_ecmult.h src/precomputed_ecmult_gen.c src/precomputed_ecmult_gen.h src/scalar.h src/scalar_4x64.h src/scalar_4x64_impl.h src/scalar_8x32.h src/scalar_8x32_impl.h src/scalar_impl.h src/scalar_low.h src/scalar_low_impl.h src/scratch.h src/scratch_impl.h src/secp256k1.c src/selftest.h src/util.h
        mv $f ../libsecp256k1/$f
    end
    cd ..
    rm -r bitcoin-core-secp256k1-*
