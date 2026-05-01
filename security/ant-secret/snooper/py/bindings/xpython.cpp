#include "xpython.h"


namespace NPySnooper::XPython
{

    namespace
    {
        TString Repr(const Py::Object& obj)
        {
            return obj.repr().as_std_string("utf-8", "replace");
        }
    }  // namespace anonymous

    Py::Object ExtractArgument(Py::Tuple& args, Py::Dict& kwargs, const std::string& name)
    {
        Py::Object result;
        if (kwargs.hasKey(name)) {
            result = kwargs.getItem(name);
            kwargs.delItem(name);
        } else {
            if (args.length() == 0) {
                throw Py::RuntimeError("Missing argument '" + name + "'");
            }
            result = args.front();
            args = args.getSlice(1, args.length());
        }
        return result;
    }

    bool HasArgument(const Py::Tuple& args, const Py::Dict& kwargs, const std::string& name)
    {
        if (kwargs.hasKey(name)) {
            return true;
        } else {
            return args.length() > 0;
        }
    }

    TString ConvertStringObjectToString(const Py::Object& obj)
    {
        Py::Object pyString = obj;
        if (!PyBytes_Check(pyString.ptr())) {
            if (PyUnicode_Check(pyString.ptr())) {
                pyString = Py::Object(PyUnicode_AsUTF8String(pyString.ptr()), true);
            } else {
                throw Py::RuntimeError("Object '" + Repr(pyString) + "' is not bytes or unicode string");
            }
        }
        char* stringData;
        Py_ssize_t length;
        PyBytes_AsStringAndSize(pyString.ptr(), &stringData, &length);
        return TString(stringData, length);
    }

    Py::Bytes ConvertToPythonString(TStringBuf string)
    {
        return Py::Bytes(string.begin(), string.length());
    }

}  // namespace NPySnooper::XPython
