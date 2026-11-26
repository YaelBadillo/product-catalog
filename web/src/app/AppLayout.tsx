import { Box, Container, Group, Text } from "@mantine/core";
import { NavLink, Outlet } from "react-router";
import { messages } from "../features/products/messages";
import classes from "./AppLayout.module.css";

const links = [
  { to: "/products", label: messages.nav.catalog, end: true },
  { to: "/products/new", label: messages.nav.newProduct, end: true },
  { to: "/status", label: messages.nav.status, end: true },
];

/** Header with the main navigation (FR-044) above every view. */
export function AppLayout() {
  return (
    <>
      <Box component="header" bg="black" c="white" py="sm">
        <Container size="lg">
          <Group justify="space-between" gap="sm">
            <Text fw={700}>{messages.appName}</Text>
            <nav aria-label={messages.nav.label}>
              <Group
                component="ul"
                gap="md"
                m={0}
                p={0}
                style={{ listStyle: "none" }}
              >
                {links.map((link) => (
                  <li key={link.to}>
                    <NavLink
                      to={link.to}
                      end={link.end}
                      className={classes.link}
                    >
                      {link.label}
                    </NavLink>
                  </li>
                ))}
              </Group>
            </nav>
          </Group>
        </Container>
      </Box>
      <Outlet />
    </>
  );
}
