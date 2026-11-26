import { Button, Container, Stack, Text, Title } from "@mantine/core";
import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { ApiError } from "../../../api/errors";
import { PageState } from "../../../components/ui/PageState";
import { useDocumentTitle } from "../../../lib/use-document-title";
import { ConflictAlert } from "../components/ConflictAlert";
import {
  ProductForm,
  type ProductFormHelpers,
} from "../components/ProductForm";
import {
  applyServerErrors,
  errorText,
  notifyError,
  notifySuccess,
} from "../feedback";
import { useProduct, useUpdateProduct } from "../hooks";
import { isProductId } from "../ids";
import { messages } from "../messages";
import { useBackToCatalog, useCatalogState } from "../navigation";
import { toFormValues, toPatchBody, type ProductFormValues } from "../schema";
import type { Product } from "../types";

function isCode(error: unknown, code: ApiError["code"]): boolean {
  return error instanceof ApiError && error.code === code;
}

function NotFound() {
  useDocumentTitle(messages.notFound.title);
  const back = useBackToCatalog();
  return (
    <PageState
      headingLevel={1}
      title={messages.notFound.title}
      text={messages.notFound.text}
      action={<Button onClick={back}>{messages.actions.backToCatalog}</Button>}
    />
  );
}

function EditForm({ id }: { id: string }) {
  const navigate = useNavigate();
  const { catalogSearch } = useCatalogState();
  const query = useProduct(id);
  const update = useUpdateProduct(id);
  // The version the user started editing. Background refetches never replace
  // it, so someone else's change surfaces as a conflict on save (FR-029).
  const [loaded, setLoaded] = useState<Product>();
  const [conflict, setConflict] = useState(false);
  const [deleted, setDeleted] = useState(false);
  if (!loaded && query.data) {
    setLoaded(query.data);
  }
  useDocumentTitle(
    loaded ? messages.pages.edit(loaded.name) : messages.actions.edit,
  );

  async function reload() {
    const { data } = await query.refetch();
    if (data) {
      setLoaded(data);
      setConflict(false);
    }
  }

  async function handleSubmit(
    values: ProductFormValues,
    { setError, dirtyFields }: ProductFormHelpers,
  ) {
    if (!loaded) {
      return;
    }
    try {
      await update.mutateAsync(
        toPatchBody(dirtyFields, values, loaded.version),
      );
      notifySuccess(messages.feedback.saved);
      await navigate(`/products/${id}`, {
        replace: true,
        state: { catalogSearch },
      });
    } catch (error) {
      if (isCode(error, "VERSION_CONFLICT")) {
        setConflict(true);
      } else if (isCode(error, "NOT_FOUND")) {
        setDeleted(true);
      } else if (!applyServerErrors(error, setError)) {
        notifyError(error);
      }
    }
  }

  if (deleted || isCode(query.error, "NOT_FOUND")) {
    return <NotFound />;
  }
  if (!loaded) {
    if (query.isPending) {
      return (
        <Text role="status" c="dimmed">
          {messages.states.loading}
        </Text>
      );
    }
    return (
      <PageState
        headingLevel={1}
        title={errorText(query.error)}
        action={
          <Button
            disabled={query.isFetching}
            onClick={() => {
              void query.refetch();
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
      <Title order={1} style={{ overflowWrap: "anywhere" }}>
        {messages.pages.edit(loaded.name)}
      </Title>
      {conflict && (
        <ConflictAlert
          reloading={query.isFetching}
          onReload={() => {
            void reload();
          }}
        />
      )}
      <ProductForm
        key={loaded.version}
        mode="edit"
        defaultValues={toFormValues(loaded)}
        isPending={update.isPending}
        onSubmit={handleSubmit}
      />
    </Stack>
  );
}

export function ProductEditPage() {
  const { id } = useParams();
  return (
    <Container component="main" size="sm" py="xl">
      {isProductId(id) ? <EditForm id={id} /> : <NotFound />}
    </Container>
  );
}
