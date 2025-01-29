#include <array>
           struct ExtensionDescriptor {
              const char* Name;
              const char* RequiredLibraries[1 + 1 + 20];
           };
           std::array<ExtensionDescriptor, 1> AvailableExtensions{
ExtensionDescriptor{"Polly", {"support", "core", "scalaropts", "instcombine", "transformutils", "analysis", "ipo", "mc", "passes", "linker", "irreader", "analysis", "bitreader", "mcparser", "object", "profiledata", "target", "targetparser", "vectorize", "PollyISL", "Polly",nullptr}},
};
