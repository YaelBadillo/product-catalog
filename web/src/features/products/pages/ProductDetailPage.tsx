import { Button, Container, Group, Stack, Text, Title } from "@mantine/core";
import { ArrowLeft } from "lucide-react";
import { useParams } from "react-router";
import { ApiError } from "../../../api/errors";
import { PageState } from "../../../components/ui/PageState";
import { useDocumentTitle } from "../../../lib/use-document-title";
import { ProductDetails } from "../components/ProductDetails";
import { errorText } from "../feedback";
import { useProduct } from "../hooks";
import { isProductId } from "../ids";
import { messages } from "../messages";
import { useBackToCatalog } from "../navigation";

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
  const {
    data: product,
    error,
    isPending,
    isFetching,
    refetch,
  } = useProduct(id);
  useDocumentTitle(product?.name ?? messages.labels.name);

  if (isPending) {
    return (
      <Text role="status" c="dimmed">
        {messages.states.loading}
      </Text>
    );
  }
  if (error instanceof ApiError && error.code === "NOT_FOUND") {
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
        <BackButton />
      </Group>
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
