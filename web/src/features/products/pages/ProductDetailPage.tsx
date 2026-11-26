import { Button, Container, Group, Stack, Text, Title } from "@mantine/core";
import { ArrowLeft, Pencil, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { ApiError } from "../../../api/errors";
import { PageState } from "../../../components/ui/PageState";
import { useDocumentTitle } from "../../../lib/use-document-title";
import { ConflictAlert } from "../components/ConflictAlert";
import { openDeleteConfirm } from "../components/delete-confirm";
import { ProductDetails } from "../components/ProductDetails";
import {
  deleteErrorText,
  errorText,
  notifyError,
  notifySuccess,
} from "../feedback";
import { useDeleteProduct, useProduct } from "../hooks";
import { isProductId } from "../ids";
import { messages } from "../messages";
import { useBackToCatalog, useCatalogState } from "../navigation";
import type { Product } from "../types";

function isCode(error: unknown, code: ApiError["code"]): boolean {
  return error instanceof ApiError && error.code === code;
}

function BackButton() {
  const back = useBackToCatalog();
  return (
    <Button
      variant="default"
      leftSection={<ArrowLeft aria-hidden="true" size={16} />}
      onClick={back}
    >
      {messages.actions.backToCatalog}
    </Button>
  );
}

function NotFound() {
  useDocumentTitle(messages.notFound.title);
  return (
    <PageState
      headingLevel={1}
      title={messages.notFound.title}
      text={messages.notFound.text}
      action={<BackButton />}
    />
  );
}

function ProductDetailContent({ id }: { id: string }) {
  const { catalogSearch } = useCatalogState();
  const navigate = useNavigate();
  const remove = useDeleteProduct();
  // A successful deletion removes this product's cache entry while the page is
  // still mounted; disabling the query keeps it from being requested again.
  const {
    data: product,
    error,
    isPending,
    isFetching,
    refetch,
  } = useProduct(id, { enabled: !remove.isPending && !remove.isSuccess });
  const [conflict, setConflict] = useState(false);

  function handleDelete(current: Product) {
    const toCatalog = () =>
      navigate(`/products${catalogSearch ?? ""}`, { replace: true });
    openDeleteConfirm(current, async () => {
      try {
        await remove.mutateAsync({ id: current.id, version: current.version });
        notifySuccess(messages.feedback.deleted);
        await toCatalog();
      } catch (deleteError) {
        if (isCode(deleteError, "NOT_FOUND")) {
          notifyError(deleteError, messages.feedback.alreadyDeleted);
          await toCatalog();
        } else if (isCode(deleteError, "VERSION_CONFLICT")) {
          setConflict(true);
        } else {
          notifyError(deleteError, deleteErrorText(deleteError));
        }
      }
    });
  }
  useDocumentTitle(product?.name ?? messages.labels.name);

  if (isPending) {
    return (
      <Text role="status" c="dimmed">
        {messages.states.loading}
      </Text>
    );
  }
  if (isCode(error, "NOT_FOUND")) {
    return <NotFound />;
  }
  if (!product) {
    return (
      <PageState
        headingLevel={1}
        title={errorText(error)}
        action={
          <Button
            disabled={isFetching}
            onClick={() => {
              void refetch();
            }}
          >
            {messages.actions.retry}
          </Button>
        }
      />
    );
  }
  return (
    <Stack gap="lg">
      <Group justify="space-between" align="flex-start" gap="sm">
        <Title order={1} style={{ overflowWrap: "anywhere" }}>
          {product.name}
        </Title>
        <Group gap="xs">
          <BackButton />
          <Button
            variant="outline"
            color="red.9"
            leftSection={<Trash2 aria-hidden="true" size={16} />}
            disabled={remove.isPending}
            onClick={() => {
              handleDelete(product);
            }}
          >
            {messages.actions.delete}
          </Button>
          <Button
            component={Link}
            to={`/products/${product.id}/edit`}
            state={{ catalogSearch }}
            leftSection={<Pencil aria-hidden="true" size={16} />}
          >
            {messages.actions.edit}
          </Button>
        </Group>
      </Group>
      {conflict && (
        <ConflictAlert
          reloading={isFetching}
          onReload={() => {
            setConflict(false);
            void refetch();
          }}
        />
      )}
      <ProductDetails product={product} />
    </Stack>
  );
}

export function ProductDetailPage() {
  const { id } = useParams();
  return (
    <Container component="main" size="md" py="xl">
      {isProductId(id) ? <ProductDetailContent id={id} /> : <NotFound />}
    </Container>
  );
}
