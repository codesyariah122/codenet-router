import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { LoginPage } from "./LoginPage";

const mocks = vi.hoisted(() => ({ login: vi.fn(), navigate: vi.fn() }));

vi.mock("../auth/AuthContext", () => ({
  useAuth: () => ({ login: mocks.login, session: null, loading: false }),
}));
vi.mock("react-router-dom", async (loadOriginal) => {
  const original = await loadOriginal<typeof import("react-router-dom")>();
  return { ...original, useNavigate: () => mocks.navigate };
});

describe("LoginPage", () => {
  beforeEach(() => {
    mocks.login.mockReset();
    mocks.navigate.mockReset();
  });

  it("mengirim kredensial dan membuka dashboard saat berhasil", async () => {
    mocks.login.mockResolvedValue(undefined);
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    );
    fireEvent.change(screen.getByLabelText("Username"), {
      target: { value: "admin" },
    });
    fireEvent.change(screen.getByLabelText("Password"), {
      target: { value: "password-test" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Masuk ke WebFig" }));
    await waitFor(() =>
      expect(mocks.login).toHaveBeenCalledWith("admin", "password-test"),
    );
    expect(mocks.navigate).toHaveBeenCalledWith("/", { replace: true });
  });

  it("menampilkan error login tanpa membocorkan password", async () => {
    mocks.login.mockRejectedValue(new Error("Login tidak dapat diproses"));
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    );
    fireEvent.change(screen.getByLabelText("Username"), {
      target: { value: "admin" },
    });
    fireEvent.change(screen.getByLabelText("Password"), {
      target: { value: "rahasia" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Masuk ke WebFig" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Login tidak dapat diproses",
    );
    expect(screen.getByRole("alert")).not.toHaveTextContent("rahasia");
  });
});
