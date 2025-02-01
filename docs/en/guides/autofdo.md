# How to use Perforator for continuous profile-guided optimization

Perforator supports generating sPGO (sampling Profile-guided optimization) profiles for subsequent use in the compilation pipeline. This guide shows how to use such profiles for performance gains and outlines the hardware/software prerequisites of deployment/toolchain for profiles generation to work.

## Toolchain prerequisites

* LLVM-based toolchain (clang/lld)
* Binary built with DWARF debug info (we recommend also using `-fdebug-info-for-profiling` clang flag for higher profile quality)

## Deployment prerequisites

* Intel CPU
* Linux Kernel >=5.7
* LBR (Last Branch Records) available for reading. Many cloud providers disable LBR by default.

## Acquiring the sPGO profile

Run the following command `./perforator/cmd/cli/perforator pgo <service-name>`, targeting your installation of Perforator.

In case of failure, recheck the deployment prerequisites and try again; in case of success, check the `TakenBranchesToExecutableBytesRatio` value in the command output: we recommend it to be at least 1.0 for higher profile quality. If the value is less than 1.0, one might either increase profile sampling rate or reschedule the deployment to have more Intel CPU + Kernel >=5.7 nodes.

If everything goes well, you now have a sPGO profile for use in subsequent compilation.

## Using the sPGO profile in the compilation pipeline

* Add the `-fprofile-sample-use=<path-to-spgo-profile>` flag to clang flags
* In case of LTO being used, add `-fprofile-sample-use=<path-to-spgo-profile>` flags to lld flags

{% note warning %}

Build caches don't necessarily play nicely with `-fprofile-sample-use`: if the path doesn't change, but the profile content changes, build cache system might consider a compilation invocation as a cache hit, even though it clearly is not, since the profile has changed.

At best, this would lead to linker errors, at worst to some kind of UB at runtime. To avoid such situations, we recommend always generating random paths to the profile.

{% endnote %}
