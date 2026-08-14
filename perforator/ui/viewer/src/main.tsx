import React from "react";
import ReactDOM from "react-dom/client";

import { configure } from "src/configure";

import { ViewerApp } from "./components/Viewer/Viewer.tsx";

configure();

ReactDOM.createRoot(document.getElementById("root")!).render(
    <React.StrictMode>
        <ViewerApp />
    </React.StrictMode>,
);
