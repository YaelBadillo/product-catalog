const uuidPattern =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** Whether `id` can be a product id; anything else is "not found" at once. */
export function isProductId(id: string | undefined): id is string {
  return id !== undefined && uuidPattern.test(id);
}
