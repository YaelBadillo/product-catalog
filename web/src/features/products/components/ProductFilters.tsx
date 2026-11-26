import { Select, SimpleGrid, TextInput } from "@mantine/core";
import { Search } from "lucide-react";
import { useState } from "react";
import { messages } from "../messages";
import type { CatalogSearch } from "../search-params";
import { sortFields, sortOrders } from "../search-params";
import { categories, currencies, type Category, type Currency } from "../types";
import { PriceInput } from "./PriceInput";

const all = "all";

const categoryOptions = [
  { value: all, label: messages.labels.all },
  ...categories.map((value) => ({ value, label: messages.categories[value] })),
];
const statusOptions = [
  { value: all, label: messages.labels.allStatuses },
  { value: "true", label: messages.statuses.active },
  { value: "false", label: messages.statuses.inactive },
];
const currencyOptions = [
  { value: all, label: messages.labels.all },
  ...currencies.map((value) => ({ value, label: value })),
];
const sortOptions = sortFields.map((value) => ({
  value,
  label: messages.sortFields[value],
}));
const orderOptions = sortOrders.map((value) => ({
  value,
  label: messages.sortOrders[value],
}));

type Price = number | "";

function toPrice(value: number | string): Price {
  return typeof value === "number" ? value : "";
}

function toOptional(value: Price): number | undefined {
  return value === "" ? undefined : value;
}

/** The rule a filter price breaks, or undefined. */
function rangeIssue(min: Price, max: Price): string | undefined {
  const tooPrecise = [min, max].some(
    (v) => v !== "" && Math.abs(v * 100 - Math.round(v * 100)) > 1e-6,
  );
  if (tooPrecise) {
    return messages.rules.priceDecimals;
  }
  if (min !== "" && max !== "" && min > max) {
    return messages.rules.priceRange;
  }
  return undefined;
}

interface ProductFiltersProps {
  search: CatalogSearch;
  onFilters: (filters: Partial<Omit<CatalogSearch, "page">>) => void;
  onSearchLater: (text: string) => void;
  onSearchNow: (text: string) => void;
  onSearchBlur: () => void;
  onPriceRangeLater: (min: number | undefined, max: number | undefined) => void;
  onPriceRangeCancel: () => void;
}

/** Search, filters, price range and sort of the catalog. */
export function ProductFilters({
  search,
  onFilters,
  onSearchLater,
  onSearchNow,
  onSearchBlur,
  onPriceRangeLater,
  onPriceRangeCancel,
}: ProductFiltersProps) {
  // Local inputs follow the URL when it changes elsewhere (Back, clear).
  const urlValues = `${search.q ?? ""}|${String(search.minPrice)}|${String(search.maxPrice)}`;
  const [syncedWith, setSyncedWith] = useState(urlValues);
  const [text, setText] = useState(search.q ?? "");
  const [minPrice, setMinPrice] = useState<Price>(search.minPrice ?? "");
  const [maxPrice, setMaxPrice] = useState<Price>(search.maxPrice ?? "");
  if (syncedWith !== urlValues) {
    setSyncedWith(urlValues);
    setText(search.q ?? "");
    setMinPrice(search.minPrice ?? "");
    setMaxPrice(search.maxPrice ?? "");
  }

  const priceError = rangeIssue(minPrice, maxPrice);
  const noCurrency = search.currency === undefined;

  function changePrices(min: Price, max: Price) {
    setMinPrice(min);
    setMaxPrice(max);
    if (rangeIssue(min, max)) {
      onPriceRangeCancel();
      return;
    }
    onPriceRangeLater(toOptional(min), toOptional(max));
  }

  return (
    <SimpleGrid
      component="section"
      aria-label={messages.labels.filters}
      cols={{ base: 1, xs: 2, md: 4 }}
      spacing="sm"
    >
      <TextInput
        type="search"
        label={messages.labels.search}
        placeholder={messages.labels.searchPlaceholder}
        leftSection={<Search aria-hidden="true" size={16} />}
        value={text}
        maxLength={100}
        onChange={(event) => {
          setText(event.currentTarget.value);
          onSearchLater(event.currentTarget.value);
        }}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            onSearchNow(event.currentTarget.value);
          }
        }}
        onBlur={onSearchBlur}
      />
      <Select
        label={messages.labels.category}
        data={categoryOptions}
        allowDeselect={false}
        value={search.category ?? all}
        onChange={(value) => {
          onFilters({
            category: value === all || !value ? undefined : (value as Category),
          });
        }}
      />
      <Select
        label={messages.labels.status}
        data={statusOptions}
        allowDeselect={false}
        value={search.isActive === undefined ? all : String(search.isActive)}
        onChange={(value) => {
          onFilters({
            isActive: value === all || !value ? undefined : value === "true",
          });
        }}
      />
      <Select
        label={messages.labels.currency}
        data={currencyOptions}
        allowDeselect={false}
        value={search.currency ?? all}
        onChange={(value) => {
          const currency =
            value === all || !value ? undefined : (value as Currency);
          onPriceRangeCancel();
          onFilters(
            currency
              ? { currency }
              : { currency, minPrice: undefined, maxPrice: undefined },
          );
        }}
      />
      <PriceInput
        label={messages.labels.minPrice}
        disabled={noCurrency}
        description={
          noCurrency ? messages.hints.priceRangeNeedsCurrency : undefined
        }
        value={minPrice}
        onChange={(value) => {
          changePrices(toPrice(value), maxPrice);
        }}
      />
      <PriceInput
        label={messages.labels.maxPrice}
        disabled={noCurrency}
        description={
          noCurrency ? messages.hints.priceRangeNeedsCurrency : undefined
        }
        value={maxPrice}
        error={priceError}
        onChange={(value) => {
          changePrices(minPrice, toPrice(value));
        }}
      />
      <Select
        label={messages.labels.sort}
        data={sortOptions}
        allowDeselect={false}
        value={search.sort}
        onChange={(value) => {
          if (value) {
            onFilters({ sort: value });
          }
        }}
      />
      <Select
        label={messages.labels.order}
        data={orderOptions}
        allowDeselect={false}
        value={search.order}
        onChange={(value) => {
          if (value) {
            onFilters({ order: value });
          }
        }}
      />
    </SimpleGrid>
  );
}
