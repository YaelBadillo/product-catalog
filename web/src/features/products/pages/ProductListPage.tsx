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
import { Link, useLocation } from "react-router";
import { PageState } from "../../../components/ui/PageState";
import { useDocumentTitle } from "../../../lib/use-document-title";
import { ProductCardList } from "../components/ProductCardList";
import { ProductFilters } from "../components/ProductFilters";
import { ProductTable } from "../components/ProductTable";
import { ResultsAnnouncer } from "../components/ResultsAnnouncer";
import { errorText } from "../feedback";
import { useProducts } from "../hooks";
import { messages } from "../messages";
import type { CatalogState } from "../navigation";
import { hasFilters } from "../search-params";
import type { ProductList } from "../types";
import {
  useCatalogSearch,
  type CatalogSearchControls,
} from "../use-catalog-search";

const paginationLabels: Record<string, string> = {
  first: "Primera página",
  previous: "Página anterior",
  next: "Página siguiente",
  last: "Última página",
};

function Results({
  list,
  controls,
}: {
  list: ProductList;
  controls: CatalogSearchControls;
}) {
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
        <ProductTable products={list.data} linkState={linkState} />
      ) : (
        <ProductCardList products={list.data} linkState={linkState} />
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
              "aria-label": paginationLabels[control],
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
  const { search } = controls;
  const { data, error, isPending, isFetching, refetch } = useProducts(search);

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
    content = <Results list={data} controls={controls} />;
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
        {content}
      </Stack>
    </Container>
  );
}
