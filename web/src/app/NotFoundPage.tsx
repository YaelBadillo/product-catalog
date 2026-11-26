import { Button, Container } from "@mantine/core";
import { Link } from "react-router";
import { PageState } from "../components/ui/PageState";
import { messages } from "../features/products/messages";
import { useDocumentTitle } from "../lib/use-document-title";

/** Unknown paths. */
export function NotFoundPage() {
  useDocumentTitle(messages.pageNotFound.title);
  return (
    <Container component="main" size="sm" py="xl">
      <PageState
        headingLevel={1}
        title={messages.pageNotFound.title}
        text={messages.pageNotFound.text}
        action={
          <Button component={Link} to="/products">
            {messages.actions.backToCatalog}
          </Button>
        }
      />
    </Container>
  );
}
