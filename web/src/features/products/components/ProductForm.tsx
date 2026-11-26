import {
  Button,
  Group,
  NumberInput,
  Select,
  SimpleGrid,
  Stack,
  Switch,
  Textarea,
  TextInput,
} from "@mantine/core";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Controller,
  useForm,
  type UseFormReturn,
  type UseFormSetError,
} from "react-hook-form";
import { messages } from "../messages";
import { productFormSchema, type ProductFormValues } from "../schema";
import { categories, currencies } from "../types";
import { PriceInput } from "./PriceInput";

export interface ProductFormHelpers {
  setError: UseFormSetError<ProductFormValues>;
  dirtyFields: UseFormReturn<ProductFormValues>["formState"]["dirtyFields"];
  reset: UseFormReturn<ProductFormValues>["reset"];
}

interface ProductFormProps {
  mode: "create" | "edit";
  defaultValues: ProductFormValues;
  isPending: boolean;
  onSubmit: (
    values: ProductFormValues,
    helpers: ProductFormHelpers,
  ) => Promise<void> | void;
}

const categoryOptions = categories.map((value) => ({
  value,
  label: messages.categories[value],
}));
const currencyOptions = currencies.map((value) => ({ value, label: value }));

/** Normalizes NumberInput's value: a number, or "" while empty. */
function toNumberValue(value: number | string): number | "" {
  if (typeof value === "number") {
    return value;
  }
  const parsed = Number(value.replaceAll(",", ""));
  return value.trim() === "" || Number.isNaN(parsed) ? "" : parsed;
}

/** The product form, shared by create and edit. */
export function ProductForm({
  mode,
  defaultValues,
  isPending,
  onSubmit,
}: ProductFormProps) {
  const form = useForm<ProductFormValues>({
    resolver: zodResolver(productFormSchema),
    defaultValues,
  });
  const {
    register,
    control,
    handleSubmit,
    setError,
    reset,
    formState: { errors, dirtyFields, isDirty },
  } = form;
  const isEdit = mode === "edit";

  return (
    <form
      noValidate
      onSubmit={(event) => {
        void handleSubmit((values) =>
          onSubmit(values, { setError, dirtyFields, reset }),
        )(event);
      }}
    >
      <Stack gap="md">
        <TextInput
          label={messages.labels.sku}
          required={!isEdit}
          readOnly={isEdit}
          description={isEdit ? messages.hints.skuReadOnly : undefined}
          error={errors.sku?.message}
          {...register("sku")}
        />
        <TextInput
          label={messages.labels.name}
          required
          error={errors.name?.message}
          {...register("name")}
        />
        <Textarea
          label={messages.labels.description}
          autosize
          minRows={3}
          error={errors.description?.message}
          {...register("description")}
        />
        <SimpleGrid cols={{ base: 1, xs: 2 }}>
          <Controller
            control={control}
            name="price"
            render={({ field }) => (
              <PriceInput
                label={messages.labels.price}
                required
                ref={field.ref}
                name={field.name}
                value={field.value}
                onBlur={field.onBlur}
                onChange={(value) => {
                  field.onChange(toNumberValue(value));
                }}
                error={errors.price?.message}
              />
            )}
          />
          <Controller
            control={control}
            name="currency"
            render={({ field }) => (
              <Select
                label={messages.labels.currency}
                required
                data={currencyOptions}
                allowDeselect={false}
                ref={field.ref}
                name={field.name}
                value={field.value}
                onBlur={field.onBlur}
                onChange={(value) => {
                  if (value) {
                    field.onChange(value);
                  }
                }}
                error={errors.currency?.message}
              />
            )}
          />
        </SimpleGrid>
        <SimpleGrid cols={{ base: 1, xs: 2 }}>
          <Controller
            control={control}
            name="stock"
            render={({ field }) => (
              <NumberInput
                label={messages.labels.stock}
                required
                inputMode="numeric"
                allowNegative={false}
                clampBehavior="none"
                ref={field.ref}
                name={field.name}
                value={field.value}
                onBlur={field.onBlur}
                onChange={(value) => {
                  field.onChange(toNumberValue(value));
                }}
                error={errors.stock?.message}
              />
            )}
          />
          <Controller
            control={control}
            name="category"
            render={({ field }) => (
              <Select
                label={messages.labels.category}
                placeholder={messages.labels.chooseCategory}
                required
                data={categoryOptions}
                ref={field.ref}
                name={field.name}
                value={field.value === "" ? null : field.value}
                onBlur={field.onBlur}
                onChange={(value) => {
                  field.onChange(value ?? "");
                }}
                error={errors.category?.message}
              />
            )}
          />
        </SimpleGrid>
        <Controller
          control={control}
          name="isActive"
          render={({ field }) => (
            <Switch
              label={messages.labels.isActive}
              ref={field.ref}
              name={field.name}
              checked={field.value}
              onBlur={field.onBlur}
              onChange={(event) => {
                field.onChange(event.currentTarget.checked);
              }}
            />
          )}
        />
        <Group justify="flex-end">
          <Button
            type="submit"
            loading={isPending}
            disabled={isEdit && !isDirty}
          >
            {isEdit ? messages.actions.save : messages.actions.create}
          </Button>
        </Group>
      </Stack>
    </form>
  );
}
