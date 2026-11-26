import { describe, expect, it } from "vitest";
import {
  defaultCatalogSearch,
  parseCatalogSearch,
  serializeCatalogSearch,
  toListQuery,
  type CatalogSearch,
} from "./search-params";

function parse(query: string) {
  return parseCatalogSearch(new URLSearchParams(query));
}

describe("parseCatalogSearch", () => {
  it("applies the defaults", () => {
    expect(parse("")).toEqual(defaultCatalogSearch);
    expect(defaultCatalogSearch).toEqual({
      page: 1,
      sort: "created_at",
      order: "desc",
    });
  });

  it("reads every parameter", () => {
    expect(
      parse(
        "page=3&category=food&is_active=false&currency=USD&min_price=100&max_price=499.90&q=caf%C3%A9&sort=price&order=asc",
      ),
    ).toEqual({
      page: 3,
      category: "food",
      isActive: false,
      currency: "USD",
      minPrice: 100,
      maxPrice: 499.9,
      q: "café",
      sort: "price",
      order: "asc",
    });
  });

  it.each([
    "page=0",
    "page=-2",
    "page=abc",
    "page=2.5",
    "category=toys",
    "is_active=yes",
    "currency=EUR",
    "sort=color",
    "order=up",
    "q=%20%20",
  ])("ignores the invalid value in %s", (query) => {
    expect(parse(query)).toEqual(defaultCatalogSearch);
  });

  it("drops a price range without a currency", () => {
    expect(parse("min_price=100&max_price=200")).toEqual(defaultCatalogSearch);
  });

  it("drops both prices when min > max", () => {
    expect(parse("currency=MXN&min_price=500&max_price=100")).toEqual({
      ...defaultCatalogSearch,
      currency: "MXN",
    });
  });

  it.each([
    "min_price=1.234",
    "min_price=abc",
    "min_price=-1",
    "min_price=1000000000",
  ])("drops the invalid price in %s", (query) => {
    expect(parse(`currency=MXN&max_price=50&${query}`)).toEqual({
      ...defaultCatalogSearch,
      currency: "MXN",
      maxPrice: 50,
    });
  });

  it("trims the search and cuts it to 100 characters", () => {
    expect(parse("q=%20%20caf%C3%A9%20%20").q).toBe("café");
    expect(parse(`q=${"a".repeat(150)}`).q).toBe("a".repeat(100));
  });
});

describe("serializeCatalogSearch", () => {
  it("omits defaults", () => {
    expect(serializeCatalogSearch(defaultCatalogSearch).toString()).toBe("");
  });

  it("round-trips a full search", () => {
    const search: CatalogSearch = {
      page: 2,
      category: "home",
      isActive: true,
      currency: "MXN",
      minPrice: 100,
      maxPrice: 499.9,
      q: "silla",
      sort: "name",
      order: "asc",
    };

    const serialized = serializeCatalogSearch(search);

    expect(serialized.toString()).toBe(
      "page=2&category=home&is_active=true&currency=MXN&min_price=100&max_price=499.9&q=silla&sort=name&order=asc",
    );
    expect(parseCatalogSearch(serialized)).toEqual(search);
  });

  it("drops prices without a currency", () => {
    expect(
      serializeCatalogSearch({
        ...defaultCatalogSearch,
        minPrice: 1,
      }).toString(),
    ).toBe("");
  });
});

describe("toListQuery", () => {
  it("converts prices to cents and always asks for 20 per page", () => {
    expect(
      toListQuery({
        page: 2,
        category: "food",
        isActive: false,
        currency: "MXN",
        minPrice: 199.9,
        maxPrice: 1299.99,
        q: "café",
        sort: "price",
        order: "asc",
      }),
    ).toEqual({
      page: 2,
      page_size: 20,
      category: "food",
      is_active: false,
      currency: "MXN",
      min_price: 19990,
      max_price: 129999,
      q: "café",
      sort: "price",
      order: "asc",
    });
  });

  it("leaves out unset filters", () => {
    expect(toListQuery(defaultCatalogSearch)).toEqual({
      page: 1,
      page_size: 20,
      sort: "created_at",
      order: "desc",
    });
  });
});
