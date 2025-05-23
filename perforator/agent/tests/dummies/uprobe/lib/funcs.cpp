#include <chrono>
#include <thread>

extern "C" {

__attribute__((noinline)) int target_function_level3(int value) {
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
    return value * 2;
}

__attribute__((noinline)) int target_function_level2(int value) {
    int result = 0;
    for (int i = 0; i < value; i++) {
        result += target_function_level3(i);
    }
    return result;
}

__attribute__((noinline)) int recursive_function(int depth) {
    if (depth <= 0) {
        return target_function_level3(depth);
    }
    return recursive_function(depth - 1) + depth;
}

__attribute__((noinline)) int lambda_caller(int value) {
    auto lambda = [](int x) {
        return target_function_level3(x);
    };

    return lambda(value) * 3;
}

}
