import type { Category } from "./types";

/**
 * Every text the products feature shows, in Mexican Spanish addressing the
 * user as tú (FR-045 to FR-047). Components never hard-code copy.
 */
export const messages = {
  appName: "Catálogo de productos",
  nav: {
    label: "Principal",
    catalog: "Catálogo",
    newProduct: "Nuevo producto",
    status: "Estado",
  },
  pages: {
    catalog: "Catálogo",
    newProduct: "Nuevo producto",
    edit: (name: string) => `Editar ${name}`,
    notFound: "Página no encontrada",
  },
  feedback: {
    created: "Producto creado",
    saved: "Cambios guardados",
    deleted: "Producto eliminado",
    skuTaken: "Ya existe un producto con este SKU",
    conflict:
      "Otra persona modificó este producto. Recarga para ver la versión actual.",
    alreadyDeleted: "Este producto ya no existe",
    deleteFailed: "No se pudo eliminar el producto. Inténtalo de nuevo.",
    unexpected: "Algo salió mal. Inténtalo de nuevo.",
    rateLimited:
      "Demasiadas solicitudes. Espera un momento e inténtalo de nuevo.",
  },
  actions: {
    save: "Guardar",
    create: "Registrar producto",
    edit: "Editar",
    delete: "Eliminar",
    cancel: "Cancelar",
    retry: "Reintentar",
    reload: "Recargar",
    backToCatalog: "Volver al catálogo",
    clearFilters: "Limpiar filtros",
    firstPage: "Ir a la primera página",
    close: "Cerrar",
  },
  deleteConfirm: {
    title: "¿Eliminar producto?",
    body: (name: string, sku: string) =>
      `Se eliminará «${name}» (${sku}). Esta acción no se puede deshacer.`,
  },
  notFound: {
    title: "Producto no encontrado",
    text: "Es posible que se haya eliminado.",
  },
  pageNotFound: {
    title: "Página no encontrada",
    text: "La dirección no existe.",
  },
  states: {
    loading: "Cargando…",
    emptyCatalog: "Aún no hay productos",
    noResults: "Ningún producto coincide con tu búsqueda o filtros",
    outOfRange: "Esta página no tiene productos",
    results: (count: number) =>
      count === 1 ? "1 producto" : `${count.toLocaleString("es-MX")} productos`,
  },
  hints: {
    priceRangeNeedsCurrency: "Elige una moneda para filtrar por precio",
    comma: "Usa punto (.) para los decimales",
    skuReadOnly: "El SKU no se puede cambiar",
  },
  noDescription: "Sin descripción",
  labels: {
    sku: "SKU",
    name: "Nombre",
    description: "Descripción",
    price: "Precio",
    currency: "Moneda",
    stock: "Inventario",
    category: "Categoría",
    status: "Estado",
    isActive: "Activo",
    createdAt: "Creado",
    updatedAt: "Última modificación",
    actions: "Acciones",
    search: "Buscar",
    searchPlaceholder: "Nombre o SKU",
    minPrice: "Precio mínimo",
    maxPrice: "Precio máximo",
    sort: "Ordenar por",
    order: "Dirección",
    all: "Todas",
    allStatuses: "Todos",
    pagination: "Paginación",
    filters: "Filtros",
    chooseCategory: "Elige una categoría",
  },
  sortFields: {
    created_at: "Fecha de creación",
    name: "Nombre",
    price: "Precio",
    stock: "Inventario",
  },
  sortOrders: { asc: "Ascendente", desc: "Descendente" },
  categories: {
    electronics: "Electrónica",
    home: "Hogar",
    food: "Alimentos",
    clothing: "Ropa",
    other: "Otra",
  } satisfies Record<Category, string>,
  statuses: { active: "Activo", inactive: "Inactivo" },
  /** Rule broken on each form field (FR-047). */
  rules: {
    skuRequired: "Escribe el SKU",
    sku: "El SKU debe tener de 4 a 32 caracteres: letras, números y guiones",
    nameRequired: "Escribe el nombre",
    name: "El nombre debe tener de 2 a 120 caracteres",
    description: "La descripción admite como máximo 2000 caracteres",
    priceRequired: "Escribe el precio",
    pricePositive: "El precio debe ser mayor que 0",
    priceDecimals: "El precio admite como máximo 2 decimales",
    priceMax: "El precio máximo es 999,999,999.99",
    stockRequired: "Escribe el inventario",
    stock: "El inventario debe ser un número entero de 0 a 2,147,483,647",
    category: "Elige una categoría",
    currency: "Elige MXN o USD",
    priceRange: "El precio mínimo no puede ser mayor que el máximo",
  },
} as const;

/** Form field of each API field, with the Spanish rule shown for a 422. */
export const serverFieldMessages = {
  sku: { field: "sku", message: messages.rules.sku },
  name: { field: "name", message: messages.rules.name },
  description: { field: "description", message: messages.rules.description },
  price_cents: {
    field: "price",
    message: "El precio debe ser mayor que 0, con 2 decimales como máximo",
  },
  currency: { field: "currency", message: messages.rules.currency },
  stock: { field: "stock", message: messages.rules.stock },
  category: { field: "category", message: messages.rules.category },
} as const;
