# Feature Specification: Product Management

**Feature Branch**: `001-product-management`

**Created**: 2026-11-25

**Status**: Draft

**Input**: User description: "Catálogo de productos: registrar, consultar, explorar, editar y eliminar productos desde la aplicación web." Provided in Spanish with four prioritized user stories, business rules, non-functional requirements and out-of-scope items, all captured below.

## Clarifications

### Session 2026-11-25

- Q: How should the search decide which products match the typed text? → A: Partial ("contains") match on name or SKU, ignoring letter case and accents ("cafe" matches "Café Molido"; "0123" matches "CAF-0123").
- Q: When the user has chosen nothing, in what order is the catalog shown and how many products per page? → A: Newest first (creation date, descending); 20 products per page, fixed size.
- Q: Besides the data captured in the form, what else does the product detail show? → A: Creation and last-updated date and time (e.g. "25 nov 2026, 14:30"); no internal identifiers.
- Q: How is a price displayed so that MXN and USD are never confused? → A: Sign, thousands separator, two decimals and the currency code at the end: "$1,299.90 MXN", "$1,299.90 USD".
- Q: Which texts do confirmations and errors use, and does the interface address the user as tú or usted? → A: Tú, with the proposed set of texts (FR-046); field errors state the broken rule (FR-047).
- Q: Is the search applied while the user types, or only on Enter or a "Buscar" button? → A: While typing, after a short pause (about 300 ms), and immediately on Enter; the address updates without adding a history entry per pause, so Back returns to the view before the search.
- Q: When sorting by name, does the catalog follow Spanish alphabetical order? → A: Yes: ignoring letter case and accents, with ñ after n ("árbol" next to "arroz"; "ñame" after "nuez").
- Q: On which screen sizes must the application be usable? → A: Desktop and tablet as the main use; on phones (from 360 px wide) everything stays usable: the list adapts to narrow screens and pages never scroll horizontally.
- Q: How many requests per minute may one connection make before the application asks to wait? → A: 300 per minute per network address (5 per second on average).
- Q: How do price fields (form and range filter) treat thousands separators and commas? → A: The field adds thousands separators as the user types ("1,299.90"); only digits and one decimal point can be typed, and a typed comma is ignored.
- Q: Does ignoring a typed comma stop `199,90` from becoming 19,990.00? → A: No: the comma is lost, but the following digits are still appended. When a comma is ignored, the field shows the hint "Usa punto (.) para los decimales", and the thousands separators make a wrong amount visible before saving (found during planning).

## User Scenarios & Testing *(mandatory)*

An administrator manages a business's product catalog. A product has a SKU (the business's code), name, optional description, price with its currency (MXN or USD), units in inventory, category (Electronics, Home, Food, Clothing or Other) and an active/inactive status. An inactive product still exists and is editable; the status only signals that it is not available for sale.

### User Story 1 - Register a product and view its detail (Priority: P1) 🎯 MVP

The administrator fills in a form to register a product and then sees its complete information, so they can build up the catalog.

**Why this priority**: Nothing else works without products. Creating one and viewing it is the smallest slice that delivers value.

**Independent Test**: Create a product with valid data, land on its detail view, and check that every field shows what was entered, with the SKU normalized and the price formatted with its currency.

**Acceptance Scenarios**:

1. **Given** the create form filled with valid data, **When** the user submits it, **Then** the product is created, a confirmation appears and the user lands on the product's detail view.
2. **Given** the user opens the create form, **When** it appears, **Then** currency is preset to MXN, status to active, and category has no preselected value.
3. **Given** a field that breaks a rule, **When** the user submits, **Then** the error appears next to that field, focus moves to the first invalid field and nothing is saved.
4. **Given** a submission the server rejects for a field rule, **When** the response arrives, **Then** each error appears next to its field and the entered values are kept.
5. **Given** a SKU already used by another non-deleted product, **When** the user submits, **Then** a clear message appears on the SKU field and the entered values are kept.
6. **Given** the SKU is entered as `  abc-123 `, **When** the product is created, **Then** it is saved and shown as `ABC-123`.
7. **Given** a price entered as `199.90` with currency MXN, **When** the product is shown, **Then** the price appears as `$199.90 MXN`.
8. **Given** a product that does not exist or was deleted, **When** the user opens its detail (for example, from a saved link), **Then** a "Producto no encontrado" (product not found) message appears with a way back to the catalog.
9. **Given** a submission in progress, **When** the user waits, **Then** the submit control is disabled and shows progress, so the product cannot be submitted twice.

---

### User Story 2 - Browse the catalog (Priority: P2)

The administrator sees the catalog as a paginated list, filters it and searches it, to find products quickly.

**Why this priority**: Once products exist, finding them is the most frequent task, and it is the entry point for editing and deleting.

**Independent Test**: With a set of products across categories, currencies and statuses, apply each filter, the search and each sort, and check the results, the URL, reload and the Back button.

**Acceptance Scenarios**:

1. **Given** products exist, **When** the user opens the catalog, **Then** they see one page of products showing SKU, name, category, price with currency, inventory and status, plus the total of matching products and page navigation.
2. **Given** the catalog, **When** the user filters by category, status and/or currency, **Then** only products matching every selected filter are shown.
3. **Given** the catalog, **When** the user types text and pauses briefly (or presses Enter), **Then** only products whose name or SKU contains that text are shown, ignoring letter case and accents.
4. **Given** the catalog, **When** the user sorts by name, price, inventory or creation date, ascending or descending, **Then** the results follow that order.
5. **Given** no currency is selected, **When** the user looks for the price range filter, **Then** it is unavailable and the interface explains that a currency must be selected first.
6. **Given** a currency is selected, **When** the user sets a minimum and/or maximum price, **Then** only products of that currency whose price is within the range (inclusive) are shown.
7. **Given** a price range is set, **When** the user clears the currency filter, **Then** the price range is removed too.
8. **Given** no currency is selected, **When** the user sorts by price, **Then** products are grouped by currency (MXN, then USD) and ordered by price within each group in the chosen direction.
9. **Given** filters, search, sort and page are applied, **When** the user reloads, shares the link or opens it in another browser, **Then** the same view is shown; **and When** the user presses Back, **Then** the previous view returns.
10. **Given** the user is on page 3, **When** they change any filter, the search or the sort, **Then** the results start again at page 1.
11. **Given** the catalog view, **Then** it shows a distinct state for each case: loading, empty catalog (with an action to create the first product), no results for the current filters (with an action to clear them), and error (with an action to retry).
12. **Given** a product in the list, **When** the user selects it, **Then** its detail view opens; **and When** they go back, **Then** the catalog returns with the same filters, search, sort and page.

---

### User Story 3 - Edit a product (Priority: P3)

The administrator edits a product to keep its information up to date.

**Why this priority**: Catalog data changes over time (price, inventory, status), but registering and finding products come first.

**Independent Test**: Edit each editable field of an existing product, save, and check the detail view; then simulate a concurrent change and check that the edit is not applied.

**Acceptance Scenarios**:

1. **Given** an existing product, **When** the user opens its edit form, **Then** every field is prefilled with current values and the SKU is shown but cannot be changed.
2. **Given** the edit form, **When** the user changes name, description, price, currency, inventory, category and/or status and saves, **Then** a confirmation appears and the detail view shows the new values.
3. **Given** a product with a description, **When** the user empties the description and saves, **Then** the product no longer has a description.
4. **Given** an invalid value, **When** the user saves, **Then** the error appears next to its field, whether the interface or the server detects it, and nothing is saved.
5. **Given** another person changed the product after the user opened the form, **When** the user saves, **Then** the change is not applied, the user is told someone else modified the product, their entered values stay on screen, and they are offered an action to reload the current version.
6. **Given** another person deleted the product after the user opened the form, **When** the user saves, **Then** a "Producto no encontrado" (product not found) message appears.
7. **Given** the form has no changes, **Then** the save action is unavailable.
8. **Given** the user changes the currency, **When** they save, **Then** the amount is kept as entered in the new currency (no conversion).

---

### User Story 4 - Delete a product (Priority: P4)

The administrator deletes a product after confirming, to remove it from the catalog without deleting it by accident.

**Why this priority**: Removing products is less frequent than the other tasks, and marking a product inactive already hides it from sale.

**Independent Test**: Delete a product from the list and from its detail, check the confirmation, the immediate removal, that it no longer appears anywhere, and that its SKU can be reused.

**Acceptance Scenarios**:

1. **Given** a product, **When** the user chooses to delete it, **Then** a confirmation identifying the product by name and SKU appears; **and When** they cancel, **Then** nothing changes.
2. **Given** the confirmation in the list, **When** the user confirms, **Then** the product disappears from the list immediately, and a confirmation appears once the deletion succeeds.
3. **Given** a confirmed deletion from the list that fails, **When** the failure is known, **Then** the product reappears in its place and the user sees a notice.
4. **Given** the confirmation in the detail view, **When** the user confirms and the deletion succeeds, **Then** the user lands on the catalog with a confirmation; if it fails, they stay on the detail view with a notice.
5. **Given** a deleted product, **Then** it does not appear in the list, search results or detail view, and a new product can be created with its SKU.
6. **Given** another person changed the product after the user loaded it, **When** the user confirms the deletion, **Then** it is not applied, the product stays (or reappears) in the list, and the user is told to reload to see the current version.
7. **Given** another person already deleted the product, **When** the user confirms the deletion, **Then** the user is told the product no longer exists and it is removed from the list.
8. **Given** the only product on the last page, **When** the user deletes it, **Then** the previous page is shown.

---

### Edge Cases

- Two users create products with the same SKU at the same time: exactly one succeeds; the other sees the SKU message.
- SKU with inner spaces, accented letters (e.g. `Ñ`) or symbols other than hyphen, after normalization: rejected on the SKU field.
- Name made of spaces only: treated as empty (required). Surrounding spaces do not count toward the 2–120 limit and are not saved.
- Description made of spaces only: treated as no description.
- Price of zero, negative, with more than two decimals or above the maximum: rejected on the price field.
- Comma typed in a price field (e.g. `199,90` meaning 199.90): the comma is not entered and the field shows the hint "Usa punto (.) para los decimales". If the user keeps typing, the thousands separators make the wrong amount (`19,990`) visible before saving.
- Inventory negative, fractional or above the maximum: rejected on the inventory field.
- Search text with special characters (`%`, `_`, quotes): matched literally.
- Price range with minimum greater than maximum: error on the range filter; the results do not change until it is fixed.
- Shared link with unknown or invalid values (unknown category, page 0, price range without currency): those values are ignored and defaults apply; no error page.
- Link to a page beyond the last one: a message says the page has no products, with an action to go to the first page.
- Products added or deleted by others while browsing: a product may shift between pages, but within one loaded set of pages, order is consistent (ties always resolved the same way).
- Too many requests (over 300 per minute from one network address): the user sees a message asking them to wait and retry; entered data is kept.
- Connection loss or unexpected server error: a generic message without technical details and a retry option; entered data is kept.
- Long values (120-character name, 2000-character description): shown in full on the detail view without breaking the layout; the list may truncate them visually.

## Requirements *(mandatory)*

### Functional Requirements

**Product data and rules**

- **FR-001**: A product MUST have: SKU, name, optional description, price, currency, inventory, category, status, creation date and last-updated date.
- **FR-002**: SKU MUST be required, normalized by removing surrounding whitespace and converting to uppercase, then contain 4–32 characters of uppercase letters A–Z, digits and hyphens only. It MUST be unique among non-deleted products and MUST NOT change after creation.
- **FR-003**: Name MUST be required, with surrounding whitespace removed, and 2–120 characters long after removal.
- **FR-004**: Description MUST be optional, with surrounding whitespace removed, and at most 2000 characters. Empty or whitespace-only means no description.
- **FR-005**: Price MUST be required, greater than zero, with at most two decimal places and at most 999,999,999.99. It is entered and displayed in currency units (e.g. `199.90`) and MUST be kept exactly, with no rounding drift. Price fields (the form and the range filter) MUST add thousands separators as the user types ("1,299.90") and accept only digits and one decimal point; a typed comma is ignored and the field shows the hint "Usa punto (.) para los decimales".
- **FR-006**: Currency MUST be MXN or USD, defaulting to MXN. Changing it MUST NOT convert the amount.
- **FR-007**: Inventory MUST be a required whole number from 0 to 2,147,483,647.
- **FR-008**: Category MUST be required and one of: Electronics, Home, Food, Clothing, Other (shown to users as Electrónica, Hogar, Alimentos, Ropa, Otra).
- **FR-009**: Status MUST be active or inactive, defaulting to active. Inactive products MUST remain visible, viewable and editable like any other.
- **FR-010**: The server MUST enforce every rule in FR-002 to FR-009 on every create and edit, regardless of the interface's own checks. The interface MUST check the same rules before submitting.
- **FR-011**: Validation errors MUST appear next to the field they refer to, whether the interface or the server detects them, and the entered values MUST be kept.
- **FR-012**: Prices MUST always be displayed with the sign, thousands separator, two decimals and the currency code at the end ("$1,299.90 MXN", "$1,299.90 USD"), so that MXN and USD amounts cannot be confused.

**Create and detail**

- **FR-013**: Users MUST be able to create a product from a form; on success the system MUST show a confirmation and open the product's detail view.
- **FR-014**: When the SKU is already in use by a non-deleted product, the system MUST reject the creation and show a clear message on the SKU field.
- **FR-015**: The detail view MUST show every attribute in FR-001: SKU, name, description (or a note that there is none), price with currency, inventory, category, status, and the creation and last-updated date and time in the user's local time (e.g. "25 nov 2026, 14:30"). It MUST NOT show internal identifiers.
- **FR-016**: Viewing a product that does not exist or was deleted MUST show a "Producto no encontrado" (product not found) message with a way back to the catalog.

**Browse**

- **FR-017**: The catalog MUST list products in pages of 20 (a fixed size the user cannot change), showing SKU, name, category, price with currency, inventory and status, plus the total number of matching products and page navigation.
- **FR-018**: Users MUST be able to filter by one category, by status (active or inactive) and by one currency; selected filters combine (all must match).
- **FR-019**: Users MUST be able to search by text. A product matches when its name or SKU contains the text anywhere (partial match), ignoring letter case and accents: "cafe" matches "Café Molido" and "0123" matches "CAF-0123". Special characters MUST be matched literally. The search MUST apply while the user types, after a short pause (about 300 ms), and immediately on Enter.
- **FR-020**: Users MUST be able to sort by name, price, inventory or creation date, ascending or descending. The default is newest first (creation date, descending). Sorting by name MUST follow Spanish alphabetical order: ignoring letter case and accents, with ñ after n ("árbol" next to "arroz"; "ñame" after "nuez"). Ties MUST be resolved consistently so that no product repeats or is skipped across pages.
- **FR-021**: A price range filter (minimum and/or maximum, inclusive) MUST be available only while a currency is selected. Clearing the currency MUST clear the range. A minimum greater than the maximum MUST be rejected with an error on the filter.
- **FR-022**: When sorting by price with no currency selected, results MUST be grouped by currency (MXN, then USD) and ordered by price within each group.
- **FR-023**: Filters, search, sort, direction and page MUST be reflected in the address, so that reloading or opening a shared link reproduces the view and Back/Forward move between views. While the user types a search, the address MUST update without adding history entries, so Back returns to the view before the search. Returning from a product's detail MUST restore the previous catalog view.
- **FR-024**: Changing any filter, the search or the sort MUST return to the first page.
- **FR-025**: Invalid or unknown values in a shared link MUST be ignored in favor of defaults, without an error page.
- **FR-026**: The catalog MUST show distinct states for loading, empty catalog (with an action to create a product), no results for the current filters or search (with an action to clear them), error (with a retry action) and a page beyond the last one (with an action to go to the first page).
- **FR-027**: Each product in the list MUST give access to its detail, edit and delete actions.

**Edit**

- **FR-028**: The edit form MUST be prefilled with the current values. The SKU MUST be shown and not be editable; every other attribute MUST be editable, and emptying the description MUST remove it. Inventory is changed here by setting its new value; there are no inventory adjustments or history.
- **FR-029**: An edit MUST be applied only if the product has not changed since the user loaded it; this is checked when saving, not while editing. Otherwise it MUST NOT be applied: the user is told someone else modified the product, their entered values stay on screen, and they are offered an action to reload the current version. A change by someone else MUST never be overwritten silently.
- **FR-030**: On a successful edit the system MUST show a confirmation and open the detail view with the updated values.
- **FR-031**: The save action MUST be unavailable while the form has no changes.

**Delete**

- **FR-032**: Users MUST be able to delete a product from the list and from its detail view, only after a confirmation that names the product (name and SKU). Cancelling MUST leave everything unchanged.
- **FR-033**: When deleting from the list, the product MUST disappear immediately on confirmation; if the deletion fails, it MUST reappear in its place and the user MUST see a notice.
- **FR-034**: When deleting from the detail view, a successful deletion MUST take the user to the catalog with a confirmation; a failed one MUST keep them on the detail view with a notice.
- **FR-035**: A deletion MUST be applied only if the product has not changed since the user loaded it; this is checked when the deletion is confirmed. Otherwise it MUST NOT be applied and the user is told to reload to see the current version.
- **FR-036**: Deletion is logical: the deleted product ceases to exist for the application. It MUST NOT appear in any list, search or detail view, its SKU MUST become available for a new product, and it cannot be recovered.
- **FR-037**: Deleting a product that someone else already deleted MUST tell the user it no longer exists and remove it from the list.

**Feedback, accessibility and resilience**

- **FR-038**: Every action MUST give visible feedback (confirmation or error), also announced to screen-reader users.
- **FR-039**: Controls that trigger an action MUST be disabled while that action is in progress, so no action is submitted twice.
- **FR-040**: Every view and flow MUST be operable with the keyboard alone and with a screen reader: every field has a label, errors are associated with their field, focus moves to the first invalid field after a failed submission, dialogs keep focus inside and return it when closed, and updated result counts are announced.
- **FR-041**: States such as active/inactive or errors MUST NOT be conveyed by color alone.
- **FR-042**: Unexpected errors MUST show a generic message without technical details and offer a retry; entered data MUST be kept.
- **FR-043**: When one network address sends more than 300 requests in a minute, the system MUST temporarily refuse further ones, and the interface MUST ask the user to wait and retry. People sharing a network address share this limit.
- **FR-044**: From any view, users MUST be able to reach the catalog and the create form.
- **FR-045**: All user-facing text (labels, messages, confirmations, errors, category and status names) MUST be in Mexican Spanish, including the existing views. Dates and amounts MUST follow Mexican Spanish formatting, and each page MUST declare its language so screen readers pronounce it correctly. Product data is shown as entered.

**Interface text**

- **FR-046**: The interface MUST address the user informally (tú) and use these texts:

  | Situation | Text |
  |---|---|
  | Product created / saved / deleted | "Producto creado" · "Cambios guardados" · "Producto eliminado" |
  | Delete confirmation | Title "¿Eliminar producto?" · "Se eliminará «{name}» ({SKU}). Esta acción no se puede deshacer." · Buttons "Eliminar" / "Cancelar" |
  | SKU already in use (on the field) | "Ya existe un producto con este SKU" |
  | Product not found | "Producto no encontrado" · "Es posible que se haya eliminado." · Action "Volver al catálogo" |
  | Concurrent change on save or delete | "Otra persona modificó este producto. Recarga para ver la versión actual." · Button "Recargar" |
  | Already deleted by someone else | "Este producto ya no existe" |
  | Deletion failed | "No se pudo eliminar el producto. Inténtalo de nuevo." |
  | Unexpected error | "Algo salió mal. Inténtalo de nuevo." · Button "Reintentar" |
  | Too many requests | "Demasiadas solicitudes. Espera un momento e inténtalo de nuevo." |
  | Empty catalog | "Aún no hay productos" · Button "Registrar producto" |
  | No results | "Ningún producto coincide con tu búsqueda o filtros" · Button "Limpiar filtros" |
  | Page beyond the last one | "Esta página no tiene productos" · Button "Ir a la primera página" |
  | Price range without a currency | "Elige una moneda para filtrar por precio" |
  | Product without description (detail) | "Sin descripción" |
  | Comma typed in a price field | "Usa punto (.) para los decimales" |

- **FR-047**: Field error messages MUST state the rule that was broken, e.g. "El SKU debe tener de 4 a 32 caracteres: letras, números y guiones".

**Screen sizes**

- **FR-048**: The application is designed mainly for desktop and tablet, but every view and flow MUST stay usable on screens from 360 px wide: the list adapts to narrow screens, and pages MUST NOT scroll horizontally.

### Key Entities

- **Product**: An item the business offers. Attributes: SKU (business code, unique among non-deleted products, immutable), name, optional description, price and its currency, inventory units, category, active/inactive status, creation and last-updated dates. Each product also carries an internal marker of its current version, used to detect changes made by someone else between loading and saving or deleting.
- **Catalog view**: The combination of filters (category, status, currency, price range), search text, sort field and direction, and page that defines what the catalog shows. It is fully expressed in the shareable address.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An administrator can register a product with every field filled in under 90 seconds.
- **SC-002**: With 10,000 products in the catalog, 95% of list, filter, search and sort results appear within 1 second.
- **SC-003**: 100% of submissions that break a business rule are rejected by the server even when the interface's checks are bypassed, and each error is tied to its field.
- **SC-004**: In every concurrent edit or delete scenario, zero changes made by one user are overwritten or deleted without the other user being told.
- **SC-005**: Every view has zero automated accessibility violations, and all five flows (create, view, browse, edit, delete) can be completed with the keyboard alone.
- **SC-006**: 100% of shared or reloaded catalog links reproduce the same filters, search, sort, page and results, given unchanged data.
- **SC-007**: Every user action ends in visible feedback (progress, then confirmation or error); none ends silently.
- **SC-008**: A price entered with up to two decimals is displayed with exactly the same amount in every view, every time.
- **SC-009**: 100% of the text users see in the application is in Spanish.
- **SC-010**: All five flows can be completed on a 360 px wide screen without horizontal page scrolling.

## Assumptions

- A single role, the administrator, manages the catalog. There is no authentication (out of scope), so anyone who can open the app can manage it.
- Users are Spanish speakers in Mexico. Amounts are entered with a dot as the decimal separator (`199.90`), as is usual in Mexico. Code and project documentation stay in English; only what the user sees is in Spanish (FR-045).
- Category and currency filters take one value each; the status filter defaults to showing both.
- Search has no minimum length; surrounding spaces in the search text are ignored.
- The price maximum (999,999,999.99) and inventory maximum (2,147,483,647) are safe technical bounds, not business rules.
- When sorting by price without a currency, groups follow a fixed order (MXN, then USD) regardless of direction.
- Leaving a form with unsaved changes shows no warning.
- The existing service status view stays available.
- Platform protections deferred to this first feature (request rate limiting, request size limits, security headers, browsable API documentation) are planned with it; this spec covers only their user-visible effect (FR-043).
- Out of scope: authentication, product images, inventory adjustments with history, recovering deleted products, category management, currency conversion, bulk actions and import/export.
