import { Container, Stack, Title } from "@mantine/core";
import { useNavigate } from "react-router";
import { useDocumentTitle } from "../../../lib/use-document-title";
import {
  ProductForm,
  type ProductFormHelpers,
} from "../components/ProductForm";
import { applyServerErrors, notifyError, notifySuccess } from "../feedback";
import { useCreateProduct } from "../hooks";
import { messages } from "../messages";
import {
  defaultFormValues,
  toCreateBody,
  type ProductFormValues,
} from "../schema";

export function ProductCreatePage() {
  useDocumentTitle(messages.pages.newProduct);
  const navigate = useNavigate();
  const create = useCreateProduct();

  async function handleSubmit(
    values: ProductFormValues,
    { setError }: ProductFormHelpers,
  ) {
    try {
      const product = await create.mutateAsync(toCreateBody(values));
      notifySuccess(messages.feedback.created);
      await navigate(`/products/${product.id}`);
    } catch (error) {
      if (!applyServerErrors(error, setError)) {
        notifyError(error);
      }
    }
  }

  return (
    <Container component="main" size="sm" py="xl">
      <Stack gap="lg">
        <Title order={1}>{messages.pages.newProduct}</Title>
        <ProductForm
          mode="create"
          defaultValues={defaultFormValues}
          isPending={create.isPending}
          onSubmit={handleSubmit}
        />
      </Stack>
    </Container>
  );
}
