import { BrowserRouter, NavLink, Route, Routes } from 'react-router-dom';
import { HomePage } from './HomePage';
import { HealthPage } from './HealthPage';

export function App() {
  return (
    <BrowserRouter>
      <div className="shell">
        <header className="shell__header">
          <strong>easy-im</strong>
          <nav className="shell__nav">
            <NavLink to="/" end>
              Home
            </NavLink>
            <NavLink to="/health">API health</NavLink>
          </nav>
        </header>
        <main className="shell__main">
          <Routes>
            <Route path="/" element={<HomePage />} />
            <Route path="/health" element={<HealthPage />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}
