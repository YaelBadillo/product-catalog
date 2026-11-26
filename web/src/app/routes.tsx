import { Navigate, type RouteObject } from "react-router";
import { StatusPage } from "../features/status/pages/StatusPage";
import { AppLayout } from "./AppLayout";
import { NotFoundPage } from "./NotFoundPage";

/** Every route of the app; App and the tests' renderRoute share it. */
export const routes: RouteObject[] = [
  {
    element: <AppLayout />,
    children: [
      { index: true, element: <Navigate to="/products" replace /> },
      { path: "status", element: <StatusPage /> },
      { path: "*", element: <NotFoundPage /> },
    ],
  },
];
