
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import RegisterEvent from './pages/RegisterEvent';
import TicketSuccess from './pages/TicketSuccess';

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen">
        <Routes>
          <Route path="/" element={<RegisterEvent />} />
          <Route path="/ticket/:id" element={<TicketSuccess />} />
        </Routes>
      </div>
    </BrowserRouter>
  );
}

export default App;
