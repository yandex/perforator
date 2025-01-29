extern "C" void* dlsym(void* handle, const char* symbol);

extern "C" void* dlvsym(void* handle, char* symbol, char*) {
    return dlsym(handle, symbol);
}
