#include <iostream>
#include <thread>
#include <chrono>
#include <vector>
#include <unistd.h>
#include <dlfcn.h>
#include <mutex>
#include <sys/syscall.h>

#include <library/cpp/getopt/last_getopt.h>

typedef int (*TargetFuncLevel2Type)(int);
typedef int (*RecursiveFuncType)(int);
typedef int (*LambdaCallerType)(int);

// Define gettid() using syscall
pid_t gettid() {
    return syscall(SYS_gettid);
}

__attribute__((noinline)) int target_function_level1(int value, TargetFuncLevel2Type target_function_level2) {
    return target_function_level2(value) + 100;
}

int main(int argc, const char* argv[]) {
    int freqHz = 1;

    NLastGetopt::TOpts opts;
    opts.AddLongOption("freq-hz", "Call frequency per second (Hz)")
        .RequiredArgument("N")
        .DefaultValue("1")
        .StoreResult(&freqHz);
    opts.SetFreeArgsMin(1);
    opts.SetFreeArgsMax(1);

    NLastGetopt::TOptsParseResult parseResult(&opts, argc, argv);
    const auto freeArgs = parseResult.GetFreeArgs();

    if (freqHz <= 0) {
        std::cerr << "freq-hz must be positive" << std::endl;
        return 1;
    }

    const auto interval = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::duration<double>(1.0 / static_cast<double>(freqHz))
    );

    const char* libPath = freeArgs[0].c_str();

    std::cout << "Uprobe test program started" << std::endl;
    std::cout << "Process ID: " << getpid() << std::endl;

    void* lib_handle = dlopen(libPath, RTLD_LAZY);
    if (!lib_handle) {
        std::cerr << "Error loading library: " << dlerror() << std::endl;
        return 1;
    }

    auto target_function_level2 = (TargetFuncLevel2Type)dlsym(lib_handle, "target_function_level2");
    auto recursive_function = (RecursiveFuncType)dlsym(lib_handle, "recursive_function");
    auto lambda_caller = (LambdaCallerType)dlsym(lib_handle, "lambda_caller");

    const char* dlsym_error = dlerror();
    if (dlsym_error) {
        std::cerr << "Error resolving symbols: " << dlsym_error << std::endl;
        dlclose(lib_handle);
        return 1;
    }

    std::mutex print_mutex;
    auto thread_func = [&](int thread_num) {
        {
            std::lock_guard<std::mutex> lock(print_mutex);
            std::cout << "Thread " << thread_num << " started. "
                     << "PID: " << getpid()
                     << " TID: " << gettid() << std::endl;
        }

        int iteration = 0;
        while (true) {
            iteration++;

            int value = iteration % 5 + 1;
            target_function_level1(value, target_function_level2);
            recursive_function(value);
            lambda_caller(value);

            std::this_thread::sleep_for(interval);
        }
    };

    std::vector<std::thread> threads;
    for (int i = 0; i < 3; i++) {
        threads.emplace_back(thread_func, i);
    }

    for (auto& t : threads) {
        t.join();
    }

    dlclose(lib_handle);
    return 0;
}
