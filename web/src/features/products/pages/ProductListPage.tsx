import {
  Button,
  Container,
  Group,
  Pagination,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { Plus } from "lucide-react";
import { useState } from "react";
import { Link, useLocation } from "react-router";
import { ApiError } from "../../../api/errors";
import { PageState } from "../../../components/ui/PageState";
import { useDocumentTitle } from "../../../lib/use-document-title";
import { ConflictAlert } from "../components/ConflictAlert";
import { openDeleteConfirm } from "../components/delete-confirm";
import { ProductCardList } from "../components/ProductCardList";
import { ProductFilters } from "../components/ProductFilters";
import { ProductTable } from "../components/ProductTable";
import { ResultsAnnouncer } from "../components/ResultsAnnouncer";
import { RowActions } from "../components/RowActions";
import {
  deleteErrorText,
  errorText,
  notifyError,
  notifySuccess,
} from "../feedback";
import { useDeleteProduct, useProducts } from "../hooks";
import { messages } from "../messages";
import type { CatalogState } from "../navigation";
import { hasFilters } from "../search-params";
import type { Product, ProductList } from "../types";
import {
  useCatalogSearch,
  type CatalogSearchControls,
} from "../use-catalog-search";

interface ResultsProps {
  list: ProductList;
  controls: CatalogSearchControls;
  onDelete: (product: Product) => void;
}

function Results({ list, controls, onDelete }: ResultsProps) {
  // Read synchronously: the SPA has no server render, so no layout flash.
  const wide = useMediaQuery("(min-width: 48em)", undefined, {
    getInitialValueInEffect: false,
  });
  const location = useLocation();
  const linkState: CatalogState = {
    fromCatalog: true,
    catalogSearch: location.search,
  };
  const { search, setPage, clearFilters } = controls;
  const renderActions = (product: Product) => (
    <RowActions product={product} linkState={linkState} onDelete={onDelete} />
  );

  if (list.meta.total === 0) {
    return hasFilters(search) ? (
      <PageState
        title={messages.states.noResults}
        action={
          <Button onClick={clearFilters}>
            {messages.actions.clearFilters}
          </Button>
        }
      />
    ) : (
      <PageState
        title={messages.states.emptyCatalog}
        action={
          <Button component={Link} to="/products/new">
            {messages.actions.create}
          </Button>
        }
      />
    );
  }
  if (list.data.length === 0) {
    return (
      <PageState
        title={messages.states.outOfRange}
        action={
          <Button
            onClick={() => {
              setPage(1);
            }}
          >
            {messages.actions.firstPage}
          </Button>
        }
      />
    );
  }
  return (
    <Stack gap="md">
      {wide ? (
        <ProductTable
          products={list.data}
          linkState={linkState}
          renderActions={renderActions}
        />
      ) : (
        <ProductCardList
          products={list.data}
          linkState={linkState}
          renderActions={renderActions}
        />
      )}
      {list.meta.total_pages > 1 && (
        <Group justify="center">
          <Pagination
            component="nav"
            aria-label={messages.labels.pagination}
            total={list.meta.total_pages}
            value={search.page}
            onChange={setPage}
            getControlProps={(control) => ({
              "aria-label": messages.labels.paginationControls[control],
            })}
            getItemProps={(page) => ({ "aria-label": String(page) })}
          />
        </Group>
      )}
    </Stack>
  );
}

export function ProductListPage() {
  useDocumentTitle(messages.pages.catalog);
  const controls = useCatalogSearch();
  const { search, setPage } = controls;
  const { data, error, isPending, isFetching, refetch } = useProducts(search);
  const remove = useDeleteProduct();
  const [conflict, setConflict] = useState(false);

  function handleDelete(product: Product) {
    // Deleting the only product of a page beyond the first shows the previous one.
    const emptiesPage = data?.data.length === 1 && search.page > 1;
    openDeleteConfirm(product, async () => {
      try {
        await remove.mutateAsync({ id: product.id, version: product.version });
        notifySuccess(messages.feedback.deleted);
        if (emptiesPage) {
          setPage(search.page - 1, { replace: true });
        }
      } catch (error) {
        if (error instanceof ApiError && error.code === "NOT_FOUND") {
          notifyError(error, messages.feedback.alreadyDeleted);
        } else if (
          error instanceof ApiError &&
          error.code === "VERSION_CONFLICT"
        ) {
          setConflict(true);
        } else {
          notifyError(error, deleteErrorText(error));
        }
      }
    });
  }

  let content;
  if (isPending) {
    content = (
      <Text role="status" c="dimmed">
        {messages.states.loading}
      </Text>
    );
  } else if (error) {
    content = (
      <PageState
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
  } else {
    content = (
      <Results list={data} controls={controls} onDelete={handleDelete} />
    );
  }

  return (
    <Container component="main" size="lg" py="xl">
      <Stack gap="lg">
        <Group justify="space-between" align="center">
          <Title order={1}>{messages.pages.catalog}</Title>
          <Button
            component={Link}
            to="/products/new"
            leftSection={<Plus aria-hidden="true" size={16} />}
          >
            {messages.nav.newProduct}
          </Button>
        </Group>
        <ProductFilters
          search={search}
          onFilters={controls.setFilters}
          onSearchLater={controls.searchLater}
          onSearchNow={controls.searchNow}
          onSearchBlur={controls.endTypingSession}
          onPriceRangeLater={controls.setPriceRangeLater}
          onPriceRangeCancel={controls.cancelPriceRange}
        />
        <ResultsAnnouncer total={error ? undefined : data?.meta.total} />
        {conflict && (
          <ConflictAlert
            reloading={isFetching}
            onReload={() => {
              setConflict(false);
              void refetch();
            }}
          />
        )}
        {content}
      </Stack>
    </Container>
  );
}
