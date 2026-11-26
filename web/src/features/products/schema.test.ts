import { describe, expect, it } from "vitest";
import { messages } from "./messages";
import {
  defaultFormValues,
  productFormSchema,
  toCreateBody,
  toPatchBody,
  type ProductFormValues,
} from "./schema";

function valid(): ProductFormValues {
  return {
    sku: "ABC-123",
    name: "Café molido",
    description: "",
    price: 199.9,
    currency: "MXN",
    stock: 5,
    category: "food",
    isActive: true,
  };
}

/** The message of the first issue on `field`, or undefined. */
function errorOn(values: ProductFormValues, field: keyof ProductFormValues) {
  const result = productFormSchema.safeParse(values);
  if (result.success) {
    return undefined;
  }
  return result.error.issues.find((issue) => issue.path[0] === field)?.message;
}

describe("productFormSchema", () => {
  it("accepts valid values", () => {
    expect(productFormSchema.safeParse(valid()).success).toBe(true);
  });

  it("starts with MXN, active and no category", () => {
    expect(defaultFormValues).toMatchObject({
      currency: "MXN",
      isActive: true,
      category: "",
    });
  });

  it.each(["abc-123", "  abc-123 ", "AB-1", "A".repeat(32)])(
    "accepts the SKU %j after normalizing",
    (sku) => {
      expect(errorOn({ ...valid(), sku }, "sku")).toBeUndefined();
    },
  );

  it.each([
    ["", messages.rules.skuRequired],
    ["   ", messages.rules.skuRequired],
    ["ABC", messages.rules.sku],
    ["A".repeat(33), messages.rules.sku],
    ["ab c1", messages.rules.sku],
    ["AÑO-1", messages.rules.sku],
    ["ABC_123", messages.rules.sku],
  ])("rejects the SKU %j", (sku, message) => {
    expect(errorOn({ ...valid(), sku }, "sku")).toBe(message);
  });

  it("counts the name length after trimming", () => {
    expect(
      errorOn({ ...valid(), name: `  ${"ñ".repeat(120)}  ` }, "name"),
    ).toBeUndefined();
    expect(errorOn({ ...valid(), name: " a " }, "name")).toBe(
      messages.rules.name,
    );
    expect(errorOn({ ...valid(), name: "a".repeat(121) }, "name")).toBe(
      messages.rules.name,
    );
    expect(errorOn({ ...valid(), name: "   " }, "name")).toBe(
      messages.rules.nameRequired,
    );
  });

  it("limits the description to 2000 characters after trimming", () => {
    expect(
      errorOn(
        { ...valid(), description: ` ${"d".repeat(2000)} ` },
        "description",
      ),
    ).toBeUndefined();
    expect(
      errorOn({ ...valid(), description: "d".repeat(2001) }, "description"),
    ).toBe(messages.rules.description);
  });

  it.each([
    ["", messages.rules.priceRequired],
    [0, messages.rules.pricePositive],
    [-1, messages.rules.pricePositive],
    [199.999, messages.rules.priceDecimals],
    [1_000_000_000, messages.rules.priceMax],
    [999_999_999.991, messages.rules.priceDecimals],
  ] as const)("rejects the price %j", (price, message) => {
    expect(errorOn({ ...valid(), price }, "price")).toBe(message);
  });

  it.each([0.01, 199.9, 1299.9, 0.1 + 0.2 - 0.3 + 0.07, 999_999_999.99])(
    "accepts the price %j",
    (price) => {
      expect(errorOn({ ...valid(), price }, "price")).toBeUndefined();
    },
  );

  it.each([
    ["", messages.rules.stockRequired],
    [-1, messages.rules.stock],
    [1.5, messages.rules.stock],
    [2_147_483_648, messages.rules.stock],
  ] as const)("rejects the stock %j", (stock, message) => {
    expect(errorOn({ ...valid(), stock }, "stock")).toBe(message);
  });

  it.each([0, 2_147_483_647])("accepts the stock %j", (stock) => {
    expect(errorOn({ ...valid(), stock }, "stock")).toBeUndefined();
  });

  it("requires a category", () => {
    expect(errorOn({ ...valid(), category: "" }, "category")).toBe(
      messages.rules.category,
    );
  });
});

describe("toCreateBody", () => {
  it("trims, uppercases the SKU and converts the price to cents", () => {
    expect(
      toCreateBody({
        ...valid(),
        sku: "  abc-123 ",
        name: "  Café molido ",
        description: "  Tostado  ",
        price: 1299.9,
        currency: "USD",
        isActive: false,
      }),
    ).toEqual({
      sku: "ABC-123",
      name: "Café molido",
      description: "Tostado",
      price_cents: 129990,
      currency: "USD",
      stock: 5,
      category: "food",
      is_active: false,
    });
  });

  it("omits an empty description", () => {
    expect(toCreateBody({ ...valid(), description: "   " })).not.toHaveProperty(
      "description",
    );
  });

  it("rounds floating point noise to whole cents", () => {
    expect(toCreateBody({ ...valid(), price: 0.1 + 0.2 }).price_cents).toBe(30);
    expect(toCreateBody({ ...valid(), price: 19.99 }).price_cents).toBe(1999);
  });
});

describe("toPatchBody", () => {
  it("sends only the changed fields and the version", () => {
    expect(
      toPatchBody(
        { stock: true, isActive: true },
        { ...valid(), stock: 7, isActive: false },
        3,
      ),
    ).toEqual({ version: 3, stock: 7, is_active: false });
  });

  it("sends every editable field in API names, the price in cents", () => {
    expect(
      toPatchBody(
        {
          name: true,
          description: true,
          price: true,
          currency: true,
          stock: true,
          category: true,
          isActive: true,
        },
        {
          ...valid(),
          name: "  Té verde ",
          description: "  Hojas  ",
          price: 1299.9,
          currency: "USD",
          category: "other",
        },
        1,
      ),
    ).toEqual({
      version: 1,
      name: "Té verde",
      description: "Hojas",
      price_cents: 129990,
      currency: "USD",
      stock: 5,
      category: "other",
      is_active: true,
    });
  });

  it("sends an emptied description as an empty string", () => {
    expect(
      toPatchBody({ description: true }, { ...valid(), description: "  " }, 2),
    ).toEqual({
      version: 2,
      description: "",
    });
  });

  it("never sends the SKU", () => {
    expect(toPatchBody({ sku: true }, valid(), 1)).toEqual({ version: 1 });
  });
});
