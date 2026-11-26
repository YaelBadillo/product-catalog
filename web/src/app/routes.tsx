import { Navigate, type RouteObject } from "react-router";
import { ProductCreatePage } from "../features/products/pages/ProductCreatePage";
import { ProductDetailPage } from "../features/products/pages/ProductDetailPage";
import { StatusPage } from "../features/status/pages/StatusPage";
import { AppLayout } from "./AppLayout";
import { NotFoundPage } from "./NotFoundPage";

/** Every route of the app; App and the tests' renderRoute share it. */
export const routes: RouteObject[] = [
  {
    element: <AppLayout />,
    children: [
      { index: true, element: <Navigate to="/products" replace /> },
      { path: "products/new", element: <ProductCreatePage /> },
      { path: "products/:id", element: <ProductDetailPage /> },
      { path: "status", element: <StatusPage /> },
      { path: "*", element: <NotFoundPage /> },
    ],
  },
];
