#include <perforator/lib/tls/variable.h>

#include <thread>

#include <util/datetime/base.h>
#include <util/string/cast.h>


Y_PERFORATOR_THREAD_LOCAL_UI64(ui);
Y_PERFORATOR_THREAD_LOCAL_STRING(aboba);
Y_PERFORATOR_THREAD_LOCAL_STRINGBUF(strbuf);

thread_local int tlsvar = 5;

int main() {
    std::thread th1([]() {
        for (ui64 i = 0; true; ++i) {
            Y_PERFORATOR_GET_THREAD_LOCAL(ui).Set(i % 3 + 100);
            tlsvar = 12345678;
            if (i % 79 == 0) {
                Y_PERFORATOR_GET_THREAD_LOCAL(aboba).Set("bar");
            }
            if (i % 1001 == 0) {
                Y_PERFORATOR_GET_THREAD_LOCAL(strbuf).Set("bar");
            }
        }
    });

    std::thread th2([]() {
        for (ui64 i = 0; true; ++i) {
            Y_PERFORATOR_GET_THREAD_LOCAL(ui).Set(i % 3 + 1000);
            tlsvar = 87654321;
            if (i % 79 == 0) {
                Y_PERFORATOR_GET_THREAD_LOCAL(aboba).Set("foo");
            }
            if (i % 1337 == 0) {
                Y_PERFORATOR_GET_THREAD_LOCAL(strbuf).Set("foo");
            }
            if (i % 15 == 0) {
                Y_PERFORATOR_GET_THREAD_LOCAL(strbuf).Clear();
            }
        }
    });

    for (ui64 i = 0; true; ++i) {
        tlsvar *= tlsvar;
        Y_PERFORATOR_GET_THREAD_LOCAL(ui).Set(i % 3);
        Y_PERFORATOR_GET_THREAD_LOCAL(aboba).Set("main");
    }
}
