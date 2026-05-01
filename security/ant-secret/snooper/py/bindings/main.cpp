#include "snooper.h"

#if PY_MAJOR_VERSION < 3
extern "C" void initant_secret_snooper_bindings() { Y_UNUSED(NPySnooper::DoInit()); }
extern "C" void initant_secret_snooper_bindings_d() { initant_secret_snooper_bindings(); }
#else
extern "C" PyObject* PyInit_ant_secret_snooper_bindings() { return NPySnooper::DoInit(); }
extern "C" PyObject* PyInit_ant_secret_snooper_bindings_d() { return PyInit_ant_secret_snooper_bindings(); }
#endif
