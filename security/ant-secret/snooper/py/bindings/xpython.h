#pragma once

#define PY_SSIZE_T_CLEAN
#include <Python.h>

#include <CXX/Extensions.hxx> // pycxx
#include <CXX/Objects.hxx> // pycxx

#include <util/generic/string.h>
#include <util/generic/strbuf.h>


// mostly borrowed from YT

namespace NPySnooper::XPython
{
  Py::Object ExtractArgument(Py::Tuple& args, Py::Dict& kwargs, const std::string& name);
  bool HasArgument(const Py::Tuple& args, const Py::Dict& kwargs, const std::string& name);

  TString ConvertStringObjectToString(const Py::Object& obj);
  Py::Bytes ConvertToPythonString(TStringBuf string);
}
